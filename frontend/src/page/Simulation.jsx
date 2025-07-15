import { useState, useEffect, useRef } from 'react';
import { v4 as uuidv4 } from 'uuid';
import './styles/simulation.css';
import { Car } from '../page/components/Car';
import { StatsModal } from './components/StatsModal';
import { ModalInicial } from '../page/components/ModalInicial';

// Función auxiliar para traducir claves de estado/dirección a español.
const translate = (key) => {
  const translations = {
    waiting: 'En Espera',
    crossing: 'Cruzando',
    finished: 'Regresando al Puente',
    green: 'VERDE',
    red: 'ROJO',
    north: 'NORTE',
    south: 'SUR',
  };
  return translations[key?.toLowerCase()] || key?.toUpperCase() || 'N/A';
};

// Componente principal que maneja la lógica y renderizado de toda la simulación del puente.
export default function Simulation() {
  // Declaración de los estados del componente y referencias para manejar intervalos.
  const [showModal, setShowModal] = useState(true);
  const [carConfig, setCarConfig] = useState(null);
  const [cars, setCars] = useState([]);
  const [bridgeStatus, setBridgeStatus] = useState(null);
  const [isLoopingStopped, setIsLoopingStopped] = useState(false);
  const [showStatsModal, setShowStatsModal] = useState(false);
  const [carStats, setCarStats] = useState(null);
  const [crossingTime, setCrossingTime] = useState(0);
  const [restingTime, setRestingTime] = useState(0);

  const pollingIntervalRef = useRef(null);
  const heartbeatIntervalRef = useRef(null);
  const timerRef = useRef(null);

  // Efecto principal que gestiona la comunicación con el backend: inicia el sondeo y el envío de heartbeats.
  useEffect(() => {
    // Función para limpiar todos los intervalos activos.
    const cleanupIntervals = () => {
      clearInterval(pollingIntervalRef.current);
      clearInterval(heartbeatIntervalRef.current);
      clearInterval(timerRef.current);
    };

    if (showModal) {
      cleanupIntervals();
      return;
    }

    // Función que consulta periódicamente el estado de las colas, del puente y del propio vehículo.
    const fetchSimulationState = async () => {
      try {
        const [queueRes, statusRes] = await Promise.all([
          fetch('/api/queue'),
          fetch('/api/status')
        ]);
        if (!queueRes.ok || !statusRes.ok) return;

        const queueData = await queueRes.json();
        const statusData = await statusRes.json();
        setBridgeStatus(statusData);

        // Consulta el estado específico del vehículo del usuario para tener los datos más recientes.
        if (carConfig?.id) {
          const myCarRes = await fetch(`/api/vehicle/${carConfig.id}`);
          if (myCarRes.ok) {
            const myCarData = await myCarRes.json();
            setCarConfig(prev => ({ ...prev, ...myCarData }));
          }
        }

        const northQueue = queueData.north ?? [];
        const southQueue = queueData.south ?? [];
        let allVisibleCars = [...northQueue, ...southQueue];
        if (statusData.busy && statusData.current_car_id > 0) {
          const crossingCarRes = await fetch(`/api/vehicle/${statusData.current_car_id}`);
          if (crossingCarRes.ok) allVisibleCars.push(await crossingCarRes.json());
        }
        setCars(allVisibleCars.map(car => ({ ...car, spriteType: (car.id % 4) + 1 })));
      } catch (error) { console.error("Error en polling:", error); }
    };

    // Función que envía una señal de 'ping' para notificar al backend que el vehículo sigue activo.
    const sendHeartbeat = async () => {
      if (carConfig?.id) {
        try {
          const res = await fetch(`/api/vehicle/${carConfig.id}/ping`, { method: 'POST' });
          if (!res.ok) {
            console.warn("Sesión expirada. Deteniendo comunicación.");
            cleanupIntervals();
          }
        } catch (error) { console.error("Error en heartbeat:", error); }
      }
    };

    pollingIntervalRef.current = setInterval(fetchSimulationState, 1000);
    heartbeatIntervalRef.current = setInterval(sendHeartbeat, 5000);

    // Función de limpieza que se ejecuta cuando el componente se desmonta o las dependencias cambian.
    return cleanupIntervals;
  }, [showModal, carConfig?.id]);

  // Efecto para gestionar y sincronizar los temporizadores visuales basados en el estado del coche del usuario.
  useEffect(() => {
    clearInterval(timerRef.current);
    setCrossingTime(0);
    setRestingTime(0);

    const myStatus = carConfig?.status;
    const mySpeed = carConfig?.speed;
    const canRequeueAt = carConfig?.CanRequeueAt;

    // Si el coche está cruzando, inicia el temporizador de cruce.
    if (myStatus === 'crossing') {
      let timeLeft = Math.round(4 + (10 - mySpeed) / 9 * 8);
      setCrossingTime(timeLeft);
      timerRef.current = setInterval(() => {
        setCrossingTime(prev => (prev > 0 ? prev - 1 : 0));
      }, 1000);
    }
    // Si el coche está descansando, inicia el temporizador de descanso.
    else if (canRequeueAt > 0) {
      const now = Math.floor(Date.now() / 1000);
      const timeLeft = canRequeueAt - now;

      if (timeLeft > 0) {
        setRestingTime(timeLeft);
        timerRef.current = setInterval(() => {
          setRestingTime(prev => (prev > 0 ? prev - 1 : 0));
        }, 1000);
      }
    }

    return () => clearInterval(timerRef.current);
  }, [carConfig]);


useEffect(() => {
  const params = new URLSearchParams(window.location.search);
  let dir = params.get('dir');
  let vel = parseInt(params.get('vel'), 10);

  // Si vienen en URL, pero están mal, o queremos que siempre sean aleatorios:
  if (!dir || !vel || isNaN(vel)) {
    dir = Math.random() < 0.5 ? 'NORTE' : 'SUR';
    vel = Math.floor(Math.random() * 10) + 1;
  }

  // Ejecuta registro con valores (ya sean de URL o aleatorios)
  handleModalSubmit({ direccion: dir, velocidad: vel });
}, []);



  // Manejador para enviar los datos del formulario inicial al backend para registrar el vehículo.
  const handleModalSubmit = async ({ direccion, velocidad }) => {
    try {
      const response = await fetch('/api/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uuid: uuidv4(), direction: direccion.toUpperCase(), speed: velocidad }),
      });
      if (!response.ok) throw new Error('Error al registrar');

      const data = await response.json();
      setCarConfig({ ...data.car, spriteType: (data.car.id % 4) + 1 });
      setBridgeStatus(data.bridge_status);
      setShowModal(false);

      // Verificamos si este cliente fue iniciado manualmente (sin ?dir=...&vel=...)
      const params = new URLSearchParams(window.location.search);
      const isFromURL = params.has('dir') && params.has('vel');

      if (!isFromURL) {
        // Solo el cliente principal genera autos adicionales
        const numExtraCars = Math.floor(Math.random() * 5);

        for (let i = 0; i < numExtraCars; i++) {
          const randomSpeed = Math.floor(Math.random() * 10) + 1;
          const randomDirection = Math.random() < 0.5 ? 'NORTE' : 'SUR';

          window.open(
            `/simulacion?dir=${randomDirection}&vel=${randomSpeed}`,
            `_blank`,
            `width=1000,height=700`
          );
        }
      }
    } catch (error) {
      console.error("Error en registro:", error);
    }
  };



  // Manejador para notificar al backend que el vehículo del usuario debe dejar de entrar en la cola.
  const handleStopLoop = async () => {
    if (!carConfig || isLoopingStopped) return;
    try {
      await fetch(`/api/vehicle/${carConfig.id}/stop`, { method: 'POST' });
      setIsLoopingStopped(true);
    } catch (error) { console.error("Error al detener:", error); }
  };

  // Manejador para solicitar las estadísticas del vehículo al backend y mostrar el modal correspondiente.
  const handleShowStats = async () => {
    if (!carConfig) return;
    try {
      const response = await fetch(`/api/vehicle/${carConfig.id}/stats`);
      if (response.ok) {
        setCarStats(await response.json());
        setShowStatsModal(true);
      }
    } catch (error) { console.error("Error al obtener stats:", error); }
  };

  // Sección de renderizado del componente.
  return (
    <div className="container-simulacion">
      <header className="header">
        <div className="header-left">
          {/* Alterna entre el botón de 'Terminar' y 'Ver Estadísticas'. */}
          {!isLoopingStopped ? (
            <button onClick={handleStopLoop} className="stop-car-button">Terminar Simulación</button>
          ) : (
            <button onClick={handleShowStats} className="stats-button">Ver Estadísticas</button>
          )}
        </div>
        <div className="header-center">
          <h1>Puente de Una Vía</h1>
          <p>Simulación de Sistema Distribuido</p>
        </div>
      </header>

      {/* Renderiza los modales de inicio y de estadísticas si sus estados son verdaderos. */}
      {showModal && <ModalInicial onSubmit={handleModalSubmit} />}
      {showStatsModal && <StatsModal stats={carStats} carId={carConfig?.id} onClose={() => setShowStatsModal(false)} />}

      {/* Renderiza la simulación principal solo después de que el coche ha sido configurado. */}
      {carConfig && !showModal && (
        <main className="main-layout">
          <aside className="left-panel">
            <div className="panel-box">
              <h3>Tu Vehículo</h3>
              <p><strong>ID:</strong> {carConfig.id}</p>
              <div className="placeholder-sprite">
                <img src={`/car${carConfig.spriteType}.png`} alt="Icono de tu auto" className="car-icon-sprite" />
              </div>
            </div>

            <div className="panel-box state-panel">
              <h3>Estado del Vehículo</h3>
              <p><strong>Dirección:</strong> {translate(carConfig?.direction)}</p>
              <p><strong>Velocidad:</strong> {carConfig?.speed}</p>

              <p><strong>Estado:</strong>
                <span className={`status-${carConfig?.status?.toLowerCase() || 'waiting'}`}>
                  {translate(carConfig?.status)}
                </span>
              </p>

              <p>
                <strong>Tiempo de cruce:</strong>
                <span className="time-value">
                  {crossingTime > 0 ? `${crossingTime}s` : 'N/A'}
                </span>
              </p>
              <p>
                <strong>Próximo cruce en:</strong>
                <span className="time-value">
                  {restingTime > 0 ? `${restingTime}s` : 'N/A'}
                </span>
              </p>
            </div>

            <div className="panel-box">
              <h3>Estado del Puente</h3>
              <p><strong>Dirección Actual:</strong> {translate(bridgeStatus?.current_dir) || 'Libre'}</p>
              <p><strong>Vehículos Cruzando:</strong> {bridgeStatus?.busy ? 1 : 0}</p>
              <p><strong>Cola Norte:</strong> {bridgeStatus?.queue_north_size || 0}</p>
              <p><strong>Cola Sur:</strong> {bridgeStatus?.queue_south_size || 0}</p>
              <p><strong>Semáforo:</strong>
                <span className={bridgeStatus?.traffic_light === 'green' ? 'status-go' : 'status-stop'}>
                  {translate(bridgeStatus?.traffic_light)}
                </span>
              </p>
            </div>
          </aside>

          <section className="center-panel">
            <div className="simulacion-box">
              <div className="bridge-container">
                <img src="/bridge-sprite.png" alt="Puente" className="bridge-image" />
                <img src="/water-sprite.png" alt="Agua" className="water-image" />
                <div className="cars-lane">
                  {/* Mapea sobre la lista de todos los coches visibles para renderizar su componente `Car`. */}
                  {cars.map(car => (
                    <Car
                      key={car.id}
                      {...car}
                      // Pasa la duración de la animación solo al coche que está actualmente cruzando.
                      animationDuration={car.status === 'crossing' ? crossingTime : 0}
                    />
                  ))}
                </div>
              </div>
            </div>
          </section>
        </main>
      )}
    </div>
  );
}