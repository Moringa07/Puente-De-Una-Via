package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)
// Estructura de un vehículo con sus propiedades y estado.
type Car struct {
	ID               int       `json:"id"`
	UUID             string    `json:"uuid"`
	Direction        string    `json:"direction"`
	Speed            int       `json:"speed"`
	Conn             net.Conn  `json:"-"`
	Position         int       `json:"position"`
	Status           string    `json:"status"`
	IsLooping        bool      `json:"is_looping"`
	Stats            CarStats  `json:"stats"`
	LastSeen         time.Time `json:"-"`
	CanRequeueAt     int64     `json:"can_requeue_at,omitempty"`
	TimeEnteredQueue time.Time `json:"-"`
	TimeStartedCross time.Time `json:"-"`
}

// Estructura para la respuesta de la API que muestra las estadísticas de un coche.
type CarStatsResponse struct {
	TotalCrossings       int     `json:"total_crossings"`
	TotalTimeOnBridgeSec float64 `json:"total_time_on_bridge_sec"`
	AvgCrossingTimeSec   float64 `json:"avg_crossing_time_sec"`
	TotalWaitingTimeSec  float64 `json:"total_waiting_time_sec"`
	AvgWaitingTimeSec    float64 `json:"avg_waiting_time_sec"`
	TimeInBridgePercent  float64 `json:"time_in_bridge_percent"`
}

// Almacena los datos brutos de las estadísticas de un coche para cálculos internos.
type CarStats struct {
	TotalCrossings    int
	TotalTimeOnBridge time.Duration
	TotalWaitingTime  time.Duration
	TimeRegistered    time.Time
}

// Representa el estado actual y en tiempo real del puente.
type BridgeStatus struct {
	Busy           bool   `json:"busy"`
	CurrentDir     string `json:"current_dir"`
	CurrentCarID   int    `json:"current_car_id"`
	QueueNorthSize int    `json:"queue_north_size"`
	QueueSouthSize int    `json:"queue_south_size"`
	TrafficLight   string `json:"traffic_light"`
}

// Variables globales para gestionar el estado de la simulación.
var (
	// Sincroniza el acceso a las variables compartidas para evitar condiciones de carrera.
	mutex          sync.Mutex
	// Indica si el puente está ocupado por un coche.
	bridgeBusy     bool
	// Guarda la dirección del tráfico que tiene paso en el puente.
	currentDir     string
	// Apunta al coche que está cruzando el puente actualmente.
	currentCar     *Car
	// Cola de coches esperando en dirección norte.
	queueNorth     []Car
	// Cola de coches esperando en dirección sur.
	queueSouth     []Car
	// Contador para asignar un ID único a cada coche nuevo.
	carCounter     int
	// Mapa para registrar clientes y asociarlos a un ID de coche.
	clientRegistry = make(map[string]int)
	// Mapa que almacena todos los coches para ser consultados por la API.
	allCars        = make(map[int]Car)
)
// Función principal que inicia los servidores y procesos en segundo plano.
func main() {
	go startTCPServer()
	go startHTTPServer()
	go cleanupInactiveCars()

	log.Println("Servidores iniciados. Presione Ctrl+C para salir.")
	// Bloquea la rutina principal para mantener el programa activo.
	select {}
}

// Inicializa y ejecuta el servidor TCP para aceptar conexiones de los vehículos.
func startTCPServer() {
	listener, err := net.Listen("tcp", ":8050")
	if err != nil {
		log.Fatalf("Error al iniciar servidor TCP: %v", err)
	}
	defer listener.Close()
	log.Println("Servidor TCP activo en :8050")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error al aceptar conexión TCP: %v", err)
			continue
		}
		// Maneja cada conexión de cliente en una rutina concurrente separada.
		go handleClient(conn)
	}
}

// Configura las rutas de la API REST y pone en marcha el servidor HTTP.
func startHTTPServer() {
	r := mux.NewRouter()

	// Asigna las funciones manejadoras a cada ruta (endpoint) de la API.
	r.HandleFunc("/api/status", getStatusHandler).Methods("GET")
	r.HandleFunc("/api/register", registerVehicleHandler).Methods("POST")
	r.HandleFunc("/api/vehicle/{id}", getVehicleHandler).Methods("GET")
	r.HandleFunc("/api/queue", getQueueHandler).Methods("GET")
	r.HandleFunc("/api/vehicle/{id}/stop", stopVehicleLoopHandler).Methods("POST")
	r.HandleFunc("/api/vehicle/{id}/stats", getVehicleStatsHandler).Methods("GET")
	r.HandleFunc("/api/vehicle/{id}/ping", pingHandler).Methods("POST")

	// Configura los permisos de CORS (Cross-Origin Resource Sharing).
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"})

	corsRouter := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(r)

	log.Println("Servidor HTTP REST activo en :8080")
	// Inicia el servidor HTTP y detiene el programa si ocurre un error fatal.
	log.Fatal(http.ListenAndServe(":8080", corsRouter))
}

// Manejador HTTP que recibe un 'ping' de un vehículo para actualizar su estado de actividad.
func pingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	mutex.Lock()
	defer mutex.Unlock()

	if car, exists := allCars[id]; exists {
		// Actualiza la marca de tiempo para evitar que el coche sea eliminado por inactividad.
		car.LastSeen = time.Now()
		allCars[id] = car
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	} else {
		// Responde con error si el coche ya fue eliminado del sistema.
		respondWithError(w, http.StatusNotFound, "Vehículo no encontrado. La sesión ha expirado.")
	}
}

// Proceso en segundo plano que limpia periódicamente los vehículos inactivos del sistema.
func cleanupInactiveCars() {
	// Establece una ejecución cada 10 segundos.
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mutex.Lock()
		now := time.Now()
		for id, car := range allCars {
			// Comprueba si un coche (sin conexión TCP) ha estado inactivo por más de 15 segundos.
			if car.Conn == nil && now.Sub(car.LastSeen) > 15*time.Second {
				log.Printf("[Limpiador] Auto %d (UUID: %s) inactivo. Eliminando del sistema.", id, car.UUID)

				delete(allCars, id)

				// Asegura que el coche también sea eliminado de las colas de espera.
				queueNorth = removeCarFromSlice(queueNorth, id)
				queueSouth = removeCarFromSlice(queueSouth, id)
			}
		}
		mutex.Unlock()
	}
}

// Función auxiliar que busca un coche por su ID en un slice y lo elimina.
func removeCarFromSlice(slice []Car, carID int) []Car {
	for i, car := range slice {
		if car.ID == carID {
			// Retorna un nuevo slice combinando las partes antes y después del elemento a eliminar.
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// Manejador HTTP que devuelve el estado actual del puente y las colas.
func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	status := BridgeStatus{
		Busy:       bridgeBusy,
		CurrentDir: currentDir,
		// Obtiene el ID del coche actual, o 0 si no hay ninguno cruzando.
		CurrentCarID: func() int {
			if currentCar != nil {
				return currentCar.ID
			}
			return 0
		}(),
		QueueNorthSize: len(queueNorth),
		QueueSouthSize: len(queueSouth),
		TrafficLight:   "red",
	}

	// Determina si el semáforo puede estar en verde para una nueva solicitud.
	if !bridgeBusy || (currentCar != nil && currentCar.Direction == currentDir) {
		status.TrafficLight = "green"
	}

	respondWithJSON(w, http.StatusOK, status)
}

// Manejador HTTP que registra un vehículo enviado desde el frontend y lo pone en la cola para cruzar.
func registerVehicleHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Petición POST recibida en /api/register")

	var req struct {
		UUID      string `json:"uuid"`
		Direction string `json:"direction"`
		Speed     int    `json:"speed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decodificando JSON: %v", err)
		respondWithError(w, http.StatusBadRequest, "Formato inválido")
		return
	}

	log.Printf("Datos recibidos del frontend: UUID=%s, Dirección=%s, Velocidad=%d", req.UUID, req.Direction, req.Speed)

	mutex.Lock()
	// Verifica si el vehículo (por UUID) ya existe para asignarle el mismo ID.
	assignedID, exists := clientRegistry[req.UUID]
	if !exists {
		carCounter++
		assignedID = carCounter
		clientRegistry[req.UUID] = assignedID
	}

	car := Car{
		ID:        assignedID,
		UUID:      req.UUID,
		Direction: strings.ToUpper(req.Direction),
		Speed:     req.Speed,
		Status:    "waiting",
		IsLooping: true,
		Conn:      nil,
		TimeEnteredQueue: time.Now(),
		Stats: CarStats{
			TotalCrossings: 0,
			TimeRegistered: time.Now(),
		},
		LastSeen: time.Now(),
	}

	allCars[car.ID] = car
	mutex.Unlock()

	// Inicia la solicitud de cruce en segundo plano para no bloquear la respuesta HTTP.
	go requestCross(car)

	mutex.Lock()
	defer mutex.Unlock()

	trafficLight := "red"
	if !bridgeBusy || currentDir == car.Direction {
		trafficLight = "green"
	}

	response := struct {
		Car          Car          `json:"car"`
		BridgeStatus BridgeStatus `json:"bridge_status"`
	}{
		Car: car,
		BridgeStatus: BridgeStatus{
			Busy:       bridgeBusy,
			CurrentDir: currentDir,
			CurrentCarID: func() int {
				if currentCar != nil {
					return currentCar.ID
				}
				return 0
			}(),
			TrafficLight:   trafficLight,
			QueueNorthSize: len(queueNorth),
			QueueSouthSize: len(queueSouth),
		},
	}

	respondWithJSON(w, http.StatusOK, response)
}

// Manejador HTTP para obtener la información detallada de un vehículo específico por su ID.
func getVehicleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID inválido")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	car, exists := allCars[id]
	if !exists {
		respondWithError(w, http.StatusNotFound, "Vehículo no encontrado")
		return
	}

	respondWithJSON(w, http.StatusOK, car)
	log.Printf("IDs registrados: %v", allCars)
}

// Manejador HTTP que devuelve el contenido de las dos colas de espera (Norte y Sur).
func getQueueHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	response := struct {
		North []Car `json:"north"`
		South []Car `json:"south"`
	}{
		North: queueNorth,
		South: queueSouth,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// Manejador HTTP para indicar que un vehículo no debe volver a ponerse en la cola después de cruzar.
func stopVehicleLoopHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	mutex.Lock()
	defer mutex.Unlock()

	if car, exists := allCars[id]; exists {
		// Actualiza el estado del coche para detener su ciclo de cruces.
		car.IsLooping = false
		allCars[id] = car
		log.Printf("Recibida orden de detener para el auto %d.", id)
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "El vehículo se detendrá después de su próximo cruce."})
	} else {
		respondWithError(w, http.StatusNotFound, "Vehículo no encontrado")
	}
}

// Manejador HTTP que calcula y devuelve las estadísticas de rendimiento de un vehículo específico.
func getVehicleStatsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID de vehículo inválido")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	car, exists := allCars[id]
	if !exists {
		respondWithError(w, http.StatusNotFound, "Vehículo no encontrado")
		return
	}

	// Procesa las estadísticas crudas para generar una respuesta formateada.
	stats := car.Stats
	totalTime := time.Since(stats.TimeRegistered).Seconds()
	timeOnBridge := stats.TotalTimeOnBridge.Seconds()
	timeWaiting := stats.TotalWaitingTime.Seconds()

	resp := CarStatsResponse{
		TotalCrossings:       stats.TotalCrossings,
		TotalTimeOnBridgeSec: timeOnBridge,
		AvgCrossingTimeSec:   0,
		TotalWaitingTimeSec:  timeWaiting,
		AvgWaitingTimeSec:    0,
		TimeInBridgePercent:  0,
	}

	if stats.TotalCrossings > 0 {
		resp.AvgCrossingTimeSec = timeOnBridge / float64(stats.TotalCrossings)
		resp.AvgWaitingTimeSec = timeWaiting / float64(stats.TotalCrossings)
	}
	if totalTime > 0 {
		resp.TimeInBridgePercent = (timeOnBridge / totalTime) * 100
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// Función auxiliar para codificar y enviar una respuesta JSON con un código de estado específico.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// Función auxiliar que utiliza respondWithJSON para enviar un mensaje de error estandarizado.
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}
// Maneja la conexión TCP inicial de un vehículo, lo registra y solicita su cruce.
func handleClient(conn net.Conn) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Error leyendo datos del cliente:", err)
		conn.Close()
		return
	}

	parts := strings.Split(strings.TrimSpace(line), ",")
	if len(parts) != 3 {
		log.Printf("Formato incorrecto. Se esperaban 3 partes (UUID,Dir,Vel), recibido: %s", line)
		conn.Close()
		return
	}

	clientUUID := parts[0]
	direction := strings.ToUpper(parts[1])
	speed, _ := strconv.Atoi(parts[2])

	mutex.Lock()
	// Verifica si el vehículo es nuevo para asignarle un ID numérico único.
	assignedID, exists := clientRegistry[clientUUID]
	if !exists {
		carCounter++
		assignedID = carCounter
		clientRegistry[clientUUID] = assignedID
		log.Printf("Nuevo vehículo detectado (UUID: %s). Asignado ID numérico: %d", clientUUID, assignedID)
	}
	mutex.Unlock()

	car := Car{
		ID:        assignedID,
		UUID:      clientUUID,
		Direction: direction,
		Speed:     speed,
		Conn:      conn,
		Status:    "waiting",
	}

	mutex.Lock()
	allCars[car.ID] = car
	mutex.Unlock()

	log.Printf("[Auto %d] solicita cruzar desde %s", car.ID, car.Direction)
	requestCross(car)
}

// Gestiona una solicitud de cruce: da paso si el puente está libre o encola el vehículo si está ocupado.
func requestCross(car Car) {
	mutex.Lock()
	defer mutex.Unlock()

	// Comprobación de seguridad para evitar procesar un coche que ya fue eliminado.
	if _, exists := allCars[car.ID]; !exists {
		log.Printf("[Seguridad] Se intentó procesar al Auto %d, pero ya no existe en el registro. Ignorando.", car.ID)
		return
	}

	if c, exists := allCars[car.ID]; exists {
		c.Status = "waiting"
		allCars[car.ID] = c
	}

	// Si el puente no está ocupado, el coche puede cruzar inmediatamente.
	if !bridgeBusy {
		bridgeBusy = true
		currentDir = car.Direction
		currentCar = &car

		if c, exists := allCars[car.ID]; exists {
			c.Status = "crossing"
			allCars[car.ID] = c
		}

		go allowCross(car)
		return
	}

	// Si el puente está ocupado, el coche se añade a la cola correspondiente.
	if car.Direction == "NORTE" {
		queueNorth = append(queueNorth, car)
	} else {
		queueSouth = append(queueSouth, car)
	}
}
// Gestiona el proceso completo de un vehículo cruzando el puente: calcula la duración, simula el paso, actualiza estadísticas y decide si debe volver a la cola.
func allowCross(car Car) {
	if car.Conn != nil {
		fmt.Fprintf(car.Conn, "Auto %d, permiso concedido para cruzar\n", car.ID)
	} else {
		log.Printf("[Auto %d, Cliente HTTP] Permiso concedido para cruzar.", car.ID)
	}

	// Registra el momento exacto en que comienza el cruce.
	startTime := time.Now()

	mutex.Lock()
	if c, exists := allCars[car.ID]; exists {
		c.Status = "crossing"

		// Actualiza las estadísticas de tiempo de espera del coche.
		waitTime := startTime.Sub(c.TimeEnteredQueue)
		c.Stats.TotalWaitingTime += waitTime
		c.TimeStartedCross = startTime
		allCars[car.ID] = c
	}
	mutex.Unlock()

	// Calcula la duración del cruce basándose en la velocidad del coche.
	const tiempoBaseMax = 12
	const tiempoBaseMin = 4
	factorVelocidad := (10.0 - float64(car.Speed)) / 9.0
	tiempoCruceFloat := float64(tiempoBaseMin) + (float64(tiempoBaseMax-tiempoBaseMin) * factorVelocidad)
	tiempoCruceServidor := int(tiempoCruceFloat)
	tiempoCruceServidor += rand.Intn(3) - 1

	log.Printf("[Auto %d, Vel: %d] Cruzando el puente... (duración calculada: %d segundos)", car.ID, car.Speed, tiempoCruceServidor)
	// Simula el tiempo que el coche tarda en cruzar el puente.
	time.Sleep(time.Duration(tiempoCruceServidor) * time.Second)

	endTime := time.Now()

	mutex.Lock()
	defer mutex.Unlock()

	// Vuelve a verificar si el coche aún existe, ya que pudo ser eliminado mientras cruzaba.
	c, exists := allCars[car.ID]
	if !exists {
		log.Printf("[Auto %d] Terminó de cruzar pero ya fue eliminado del registro.", car.ID)
		bridgeBusy = false
		currentCar = nil
		go processQueue()
		return
	}

	c.Stats.TotalCrossings++

	// Calcula y registra el tiempo real que el coche estuvo en el puente.
	cruceReal := endTime.Sub(c.TimeStartedCross)
	c.Stats.TotalTimeOnBridge += cruceReal

	c.Status = "finished"

	// Invierte la dirección del coche para su próximo viaje si está en modo bucle.
	if c.Direction == "NORTE" {
		c.Direction = "SUR"
	} else {
		c.Direction = "NORTE"
	}

	allCars[c.ID] = c

	bridgeBusy = false
	currentCar = nil
	go processQueue()

	// Si el coche debe seguir cruzando, lo reencola después de un descanso.
	if c.IsLooping {
		go func(carToRequeue Car) {
			tiempoEspera := rand.Intn(13) + 6
			requeueTime := time.Now().Add(time.Duration(tiempoEspera) * time.Second)

			mutex.Lock()
			if car, exists := allCars[carToRequeue.ID]; exists {
				car.CanRequeueAt = requeueTime.Unix()
				allCars[car.ID] = car
			}
			mutex.Unlock()

			log.Printf("[Auto %d] Descansando por %d segundos. Podrá volver a la cola a las %s.", carToRequeue.ID, tiempoEspera, requeueTime.Format("15:04:05"))
			// Pausa para simular el descanso del coche antes de volver a la cola.
			time.Sleep(time.Duration(tiempoEspera) * time.Second)

			mutex.Lock()
			if car, exists := allCars[carToRequeue.ID]; exists {
				car.CanRequeueAt = 0
				car.TimeEnteredQueue = time.Now()
				allCars[car.ID] = car
			}
			mutex.Unlock()

			// Vuelve a solicitar el cruce para iniciar el ciclo de nuevo.
			requestCross(carToRequeue)

		}(c)
	} else {
		log.Printf("[Auto %d] Ha terminado su ciclo. Eliminando del sistema.", c.ID)
		// Si no está en bucle, se elimina permanentemente del sistema.
		delete(allCars, c.ID)
	}
}
// Revisa las colas y gestiona el paso del siguiente vehículo según la prioridad de dirección.
func processQueue() {
	mutex.Lock()
	defer mutex.Unlock()

	// Evita procesar la cola si el puente ya está ocupado, previniendo condiciones de carrera.
	if bridgeBusy {
		return
	}

	var nextCar *Car

	// Lógica de prioridad: primero coches en la dirección actual, luego cambia de sentido.
	if currentDir == "NORTE" && len(queueNorth) > 0 {
		nextCar = &queueNorth[0]
		queueNorth = queueNorth[1:]
	} else if currentDir == "SUR" && len(queueSouth) > 0 {
		nextCar = &queueSouth[0]
		queueSouth = queueSouth[1:]
	} else if len(queueNorth) > 0 {
		nextCar = &queueNorth[0]
		queueNorth = queueNorth[1:]
		currentDir = "NORTE"
	} else if len(queueSouth) > 0 {
		nextCar = &queueSouth[0]
		queueSouth = queueSouth[1:]
		currentDir = "SUR"
	}

	if nextCar != nil {
		bridgeBusy = true
		currentCar = nextCar

		// Inicia el cruce en una goroutine para no mantener el mutex bloqueado.
		go allowCross(*nextCar)
	} else {
		log.Println("Todas las colas están vacías. El puente ahora está libre.")
	}
}