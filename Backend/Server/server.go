//Servidor go run server.go

package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// El struct Car guarda tanto el ID numérico asignado como el UUID del cliente.
type Car struct {
	ID        int    // ID numérico secuencial asignado por el servidor.
	UUID      string // ID único generado por el cliente.
	Direction string
	Speed     int
	Conn      net.Conn
}

var (
	mutex sync.Mutex

	// Estado del puente
	bridgeBusy bool
	currentDir string

	// Colas de espera
	queueNorth []Car
	queueSouth []Car

	// --- NUEVOS ELEMENTOS PARA GESTIÓN DE IDs ---
	// Contador para asignar IDs secuenciales a los coches nuevos.
	carCounter int
	// Registro para mapear el UUID de un cliente a su ID numérico asignado.
	// Esto permite reconocer a un cliente aunque se reconecte.
	clientRegistry = make(map[string]int)
)

func main() {
	listener, err := net.Listen("tcp", ":8050")
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
	defer listener.Close()
	log.Println("Servidor del puente y registro de vehículos activo en el puerto 8050")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error al aceptar conexión: %v", err)
			continue
		}
		go handleClient(conn)
	}
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
	}

	log.Printf("[Auto %d] solicita cruzar desde %s", car.ID, car.Direction)
	requestCross(car)
}

func requestCross(car Car) {
	mutex.Lock()
	defer mutex.Unlock()

	if !bridgeBusy {
		bridgeBusy = true
		currentDir = car.Direction
		go allowCross(car)
		return
	}

	if car.Direction == "NORTE" {
		queueNorth = append(queueNorth, car)
	} else {
		queueSouth = append(queueSouth, car)
	}
	log.Printf("[Auto %d] encolado. Colas -> Norte: %d, Sur: %d", car.ID, len(queueNorth), len(queueSouth))
}

func allowCross(car Car) {
	defer car.Conn.Close()

	log.Printf(" [Auto %d] tiene permiso. Cruzando el puente...", car.ID)
	// ID numérico en el mensaje de permiso para que el cliente lo obtenga.
	fmt.Fprintf(car.Conn, "Auto %d, permiso concedido para cruzar", car.ID)

	// El servidor simula que el puente está ocupado por un tiempo.
	// Este es el tiempo aleatorio en el puente que afecta al sistema.
	tiempoCruceServidor := rand.Intn(10) + 2 
	time.Sleep(time.Duration(tiempoCruceServidor) * time.Second)

	log.Printf("[Auto %d] ha terminado de cruzar. Puente liberado.", car.ID)

	mutex.Lock()
	bridgeBusy = false
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