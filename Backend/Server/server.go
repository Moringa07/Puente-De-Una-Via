//Servidor que funciona como el puente de una vía
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/gorilla/mux"
)

type Car struct {
	ID        int       `json:"id"`
	UUID      string    `json:"uuid"`
	Direction string    `json:"direction"`
	Speed     int       `json:"speed"`
	Conn      net.Conn  `json:"-"`
	Position  int       `json:"position"` // Posición en la cola
	Status    string    `json:"status"`   // "waiting", "crossing", "finished"
}

type BridgeStatus struct {
	Busy           bool   `json:"busy"`
	CurrentDir     string `json:"current_dir"`
	CurrentCarID   int    `json:"current_car_id"`
	QueueNorthSize int    `json:"queue_north_size"`
	QueueSouthSize int    `json:"queue_south_size"`
	TrafficLight   string `json:"traffic_light"` // "green", "red"
}

var (
	mutex sync.Mutex
	bridgeBusy bool
	currentDir string
	currentCar *Car
	queueNorth []Car
	queueSouth []Car
	carCounter int
	clientRegistry = make(map[string]int)
	allCars    = make(map[int]Car) // Registro de todos los autos para la API
)

func main() {
	// Iniciar servidor TCP en una goroutine
	go startTCPServer()
	
	// Iniciar servidor HTTP/REST
	startHTTPServer()
}


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
		go handleClient(conn)
	}
}

func startHTTPServer() {
	r := mux.NewRouter()
	
	r.HandleFunc("/api/status", getStatusHandler).Methods("GET")
	r.HandleFunc("/api/register", registerVehicleHandler).Methods("POST")
	r.HandleFunc("/api/vehicle/{id}", getVehicleHandler).Methods("GET")
	r.HandleFunc("/api/queue", getQueueHandler).Methods("GET")
	
	log.Println("Servidor HTTP REST activo en :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Handlers HTTP
func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	
	status := BridgeStatus{
		Busy:           bridgeBusy,
		CurrentDir:     currentDir,
		CurrentCarID:   func() int { if currentCar != nil { return currentCar.ID }; return 0 }(),
		QueueNorthSize: len(queueNorth),
		QueueSouthSize: len(queueSouth),
		TrafficLight:   "red", // Se calcula después
	}
	
	// Determinar semáforo para nueva solicitud
	if !bridgeBusy || (currentCar != nil && currentCar.Direction == currentDir) {
		status.TrafficLight = "green"
	}
	
	respondWithJSON(w, http.StatusOK, status)
}

func registerVehicleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UUID      string `json:"uuid"`
		Direction string `json:"direction"`
		Speed     int    `json:"speed"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Formato inválido")
		return
	}
	
	mutex.Lock()
	defer mutex.Unlock()
	
	// Asignar ID (igual que en TCP)
	assignedID, exists := clientRegistry[req.UUID]
	if !exists {
		carCounter++
		assignedID = carCounter
		clientRegistry[req.UUID] = assignedID
	}
	
	// Crear auto (sin conexión TCP)
	car := Car{
		ID:        assignedID,
		UUID:      req.UUID,
		Direction: strings.ToUpper(req.Direction),
		Speed:     req.Speed,
		Status:    "waiting",
	}
	
	// Calcular posición en cola
	if car.Direction == "NORTE" {
		car.Position = len(queueNorth) + 1
	} else {
		car.Position = len(queueSouth) + 1
	}
	
	// Guardar referencia para la API
	allCars[car.ID] = car
	
	// Determinar semáforo
	trafficLight := "red"
	if !bridgeBusy || currentDir == car.Direction {
		trafficLight = "green"
	}
	
	// Responder con información combinada
	response := struct {
		Car          Car         `json:"car"`
		BridgeStatus BridgeStatus `json:"bridge_status"`
	}{
		Car: car,
		BridgeStatus: BridgeStatus{
			Busy:           bridgeBusy,
			CurrentDir:     currentDir,
			CurrentCarID:   func() int { if currentCar != nil { return currentCar.ID }; return 0 }(),
			TrafficLight:   trafficLight,
			QueueNorthSize: len(queueNorth),
			QueueSouthSize: len(queueSouth),
		},
	}
	
	respondWithJSON(w, http.StatusOK, response)
}

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

// Funciones auxiliares HTTP
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

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

	// --- LÓGICA DE REGISTRO DE ID ---
	mutex.Lock()
	assignedID, exists := clientRegistry[clientUUID]
	if !exists {
		// Si el UUID no existe en el registro, es un coche nuevo.
		carCounter++
		assignedID = carCounter
		clientRegistry[clientUUID] = assignedID
		log.Printf("Nuevo vehículo detectado (UUID: %s). Asignado ID numérico: %d", clientUUID, assignedID)
	}
	mutex.Unlock()
	// --- FIN LÓGICA DE REGISTRO ---

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

func requestCross(car Car) {
	mutex.Lock()
	defer mutex.Unlock()

	// Actualizar estado del auto
	if c, exists := allCars[car.ID]; exists {
		c.Status = "waiting"
		allCars[car.ID] = c
	}

	if !bridgeBusy {
		bridgeBusy = true
		currentDir = car.Direction
		currentCar = &car
		
		// Actualizar estado del auto
		if c, exists := allCars[car.ID]; exists {
			c.Status = "crossing"
			allCars[car.ID] = c
		}
		
		go allowCross(car)
		return
	}

	if car.Direction == "NORTE" {
		queueNorth = append(queueNorth, car)
	} else {
		queueSouth = append(queueSouth, car)
	}
}

func allowCross(car Car) {
	defer car.Conn.Close()

	// Notificar al cliente TCP
	fmt.Fprintf(car.Conn, "Auto %d, permiso concedido para cruzar", car.ID)

	tiempoCruceServidor := rand.Intn(10) + 2 
	time.Sleep(time.Duration(tiempoCruceServidor) * time.Second)

	mutex.Lock()
	// Actualizar estado del auto
	if c, exists := allCars[car.ID]; exists {
		c.Status = "finished"
		allCars[car.ID] = c
	}
	
	bridgeBusy = false
	currentCar = nil
	processQueue()
	mutex.Unlock()
}
func processQueue() {
	var nextCar *Car

	if currentDir == "NORTE" && len(queueNorth) > 0 {
		nextCar = &queueNorth[0]
		queueNorth = queueNorth[1:]
	} else if currentDir == "SUR" && len(queueSouth) > 0 {
		nextCar = &queueSouth[0]
		queueSouth = queueSouth[1:]
	} else if len(queueNorth) > 0 {
		nextCar = &queueNorth[0]
		queueNorth = queueNorth[1:]
	} else if len(queueSouth) > 0 {
		nextCar = &queueSouth[0]
		queueSouth = queueSouth[1:]
	}

	if nextCar != nil {
		bridgeBusy = true
		currentDir = nextCar.Direction
		go allowCross(*nextCar)
	} else {
		log.Println("Todas las colas están vacías. El puente ahora está libre.")
	}
}