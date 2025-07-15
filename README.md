# Simulador Distribuido de Puente de Una Vía

Este proyecto implementa un **sistema distribuido en Go** que simula un **puente de una sola vía**, donde vehículos se conectan al servidor, se identifican y solicitan cruzar el puente siguiendo reglas de concurrencia y gestión de colas por dirección.

---

## Contenido del Proyecto

- `server.go` → Servidor principal que:
  - Controla el acceso al puente.
  - Administra colas de espera por dirección.
  - Registra y asigna identificadores únicos a los vehículos.
  - Simula el cruce del puente de manera concurrente.
  - Expone una API REST para comunicación con el cliente web.

- **Frontend Web (en React)**:
  - Permite registrar vehículos que cruzan el puente.
  - Muestra animaciones de cruce en tiempo real.
  - Visualiza estadísticas por vehículo y el estado del puente.
  - Se comunica con el backend vía HTTP.

---

## Objetivos de la Simulación

✔ Controlar el acceso seguro al puente para evitar colisiones.  
✔ Simular vehículos cruzando en ambas direcciones.  
✔ Gestionar colas de espera diferenciadas por dirección (NORTE o SUR).  
✔ Permitir múltiples vehículos concurrentes vía navegador.  
✔ Asignar IDs persistentes a cada vehículo para su identificación continua.  
✔ Visualizar en tiempo real el estado del puente y las colas desde un panel web.

---

## Tecnologías Utilizadas

- **Backend:**
  - Go (Golang)
  - Goroutines y Mutex para concurrencia
  - API REST
  - Aleatoriedad en tiempos de cruce y espera

- **Frontend:**
  - React
  - Vite
  - REST API para integración con el backend

---

## Cómo Ejecutar el Proyecto

### 1. Iniciar el Servidor

```bash
cd Backend/server
go mod tidy
go run server.go
```
### 2 Iniciar el Frontend

```bash
cd frontend/client
npm i
npm run dev
```