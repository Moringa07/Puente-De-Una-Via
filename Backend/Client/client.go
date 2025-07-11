package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Stats struct {
	TotalCrossings    int
	TotalTimeOnBridge time.Duration
	TotalWaitingTime  time.Duration
	StartTime         time.Time
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Uso: go run client.go <servidor:puerto> <direccion> <velocidad>")
		return
	}

	servidor := os.Args[1]
	direccion := strings.ToUpper(os.Args[2])
	velocidad, _ := strconv.Atoi(os.Args[3])
	rand.Seed(time.Now().UnixNano())

	// Generar un UUID único por cliente
	uuid := fmt.Sprintf("Car-%d", time.Now().UnixNano()+int64(rand.Intn(1000)))
	fmt.Printf("Iniciando simulación para el vehículo con UUID: %s\n", uuid)

	// Canal para señales de interrupción
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Inicializar estadísticas
	stats := Stats{
		StartTime: time.Now(),
	}

	// Goroutine para manejar la señal de interrupción
	go func() {
		<-sigChan
		printStats(&stats, uuid)
		os.Exit(0)
	}()

	for {
		startRequest := time.Now()

		conn, err := net.Dial("tcp", servidor)
		if err != nil {
			fmt.Printf("[%s] Error al conectar: %v. Reintentando...\n", uuid, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Enviar UUID, dirección y velocidad al servidor
		fmt.Fprintf(conn, "%s,%s,%d\n", uuid, direccion, velocidad)

		// Espera la respuesta del servidor
		mensaje, _ := bufio.NewReader(conn).ReadString('\n')
		responseTime := time.Since(startRequest)
		stats.TotalWaitingTime += responseTime
		fmt.Printf("[%s] Mensaje del servidor: %s\n", uuid, strings.TrimSpace(mensaje))

		// Extraer el ID asignado si está en el mensaje
		if strings.Contains(mensaje, "Auto") {
			parts := strings.Split(mensaje, " ")
			if len(parts) >= 2 {
				idStr := strings.Trim(parts[1], ",")
				fmt.Printf("[%s] ID asignado: %s\n", uuid, idStr)
			}
		}

		conn.Close()

		// Simular el cruce del puente
		tiempoCruce := rand.Intn(10) + 2
		stats.TotalCrossings++
		stats.TotalTimeOnBridge += time.Duration(tiempoCruce) * time.Second
		fmt.Printf("[%s] Cruzando el puente durante %d segundos.\n", uuid, tiempoCruce)
		time.Sleep(time.Duration(tiempoCruce) * time.Second)

		// Simular tiempo aleatorio antes de volver a intentar
		tiempoEspera := rand.Intn(10) + 1
		fmt.Printf("[%s] Esperando %d segundos antes del próximo intento...\n\n", uuid, tiempoEspera)
		time.Sleep(time.Duration(tiempoEspera) * time.Second)

		// Mostrar estadísticas periódicamente
		if stats.TotalCrossings%5 == 0 {
			printStats(&stats, uuid)
		}
	}
}

func printStats(stats *Stats, uuid string) {
	totalDuration := time.Since(stats.StartTime)
	avgCrossingTime := time.Duration(0)
	avgWaitingTime := time.Duration(0)

	if stats.TotalCrossings > 0 {
		avgCrossingTime = stats.TotalTimeOnBridge / time.Duration(stats.TotalCrossings)
		avgWaitingTime = stats.TotalWaitingTime / time.Duration(stats.TotalCrossings)
	}

	fmt.Println("\n=== ESTADÍSTICAS DEL VEHÍCULO ===")
	fmt.Printf("UUID: %s\n", uuid)
	fmt.Printf("Tiempo total de simulación: %v\n", totalDuration.Round(time.Second))
	fmt.Printf("Número total de cruces: %d\n", stats.TotalCrossings)
	fmt.Printf("Tiempo total en el puente: %v\n", stats.TotalTimeOnBridge.Round(time.Second))
	fmt.Printf("Tiempo promedio por cruce: %v\n", avgCrossingTime.Round(time.Second))
	fmt.Printf("Tiempo total de espera: %v\n", stats.TotalWaitingTime.Round(time.Second))
	fmt.Printf("Tiempo promedio de espera: %v\n", avgWaitingTime.Round(time.Second))
	fmt.Printf("Porcentaje de tiempo en puente: %.1f%%\n",
		float64(stats.TotalTimeOnBridge)/float64(totalDuration)*100)
	fmt.Println("===============================\n")
}
