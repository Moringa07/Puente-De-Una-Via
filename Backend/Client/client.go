package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Uso: go run client.go <servidor:puerto> <direccion> <velocidad>")
		return
	}

	servidor := os.Args[1]
	direccion := os.Args[2]
	velocidad, _ := strconv.Atoi(os.Args[3])
	rand.Seed(time.Now().UnixNano())

	// Generar un UUID único por cliente
	uuid := fmt.Sprintf("Car-%d", time.Now().UnixNano()+int64(rand.Intn(1000)))
	fmt.Printf("Iniciando simulación para el vehículo con UUID: %s\n", uuid)

	for {
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
		fmt.Printf("[%s] Mensaje del servidor: %s\n", uuid, strings.TrimSpace(mensaje))

		conn.Close()

		// Simular el cruce del puente
		tiempoCruce := rand.Intn(10) + 2 
		fmt.Printf("[%s] Cruzando el puente durante %d segundos.\n", uuid, tiempoCruce)
		time.Sleep(time.Duration(tiempoCruce) * time.Second)

		// Simular tiempo aleatorio antes de volver a intentar
		tiempoEspera := rand.Intn(10) + 1 
		fmt.Printf("[%s] Esperando %d segundos antes del próximo intento...\n\n", uuid, tiempoEspera)
		time.Sleep(time.Duration(tiempoEspera) * time.Second)
	}
}
