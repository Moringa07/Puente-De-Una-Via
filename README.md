
# Simulador Distribuido de Puente de Una Vía

Este proyecto implementa un **sistema distribuido en Go** que simula un **puente de una sola vía**, donde vehículos (clientes) se conectan al servidor, se identifican y solicitan cruzar el puente siguiendo reglas de concurrencia y gestión de colas por dirección.

---

## Contenido del Proyecto

- `server.go` → Servidor principal que:
  - Controla el acceso al puente.
  - Administra colas de espera por dirección.
  - Registra y asigna identificadores únicos a los vehículos.
  - Simula el cruce del puente de manera concurrente.

- `client.go` → Cliente que representa un vehículo:
  - Genera un UUID único.
  - Se conecta al servidor.
  - Solicita cruzar el puente de forma cíclica.
  - Simula el cruce y el tiempo de espera entre intentos.

---

##  Objetivos de la Simulación

✔ Controlar el acceso seguro al puente para evitar colisiones.  
✔ Simular vehículos cruzando en ambas direcciones.  
✔ Gestionar colas de espera diferenciadas por dirección (NORTE o SUR).  
✔ Permitir múltiples clientes remotos de forma concurrente.  
✔ Asignar IDs persistentes a cada vehículo para su identificación continua.  
✔ Visualizar en tiempo real el estado del puente y las colas desde un panel web. 

---

## Tecnologías Utilizadas

- **Backend:**
  - Go (Golang)
  - Goroutines y Mutex para concurrencia
  - Red TCP para comunicación cliente-servidor
  - Aleatoriedad en tiempos de cruce y espera

- **Frontend:**
  - React
  - Vite
  - REST API

---

## Cómo Ejecutar el Proyecto

### 1️⃣ Iniciar el Servidor

```bash
cd Backend/server
go run server.go
```