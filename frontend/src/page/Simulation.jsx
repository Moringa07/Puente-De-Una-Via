import { useState, useEffect, useRef } from 'react';
import { v4 as uuidv4 } from 'uuid';
import './styles/simulation.css';
import { Car } from '../page/components/Car';
import { StatsModal } from './components/StatsModal';

// Traduce claves de estado a texto legible para el usuario.
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

// Componente principal que renderiza y gestiona la simulación.
export default function Simulation() {

  const [carConfig, setCarConfig] = useState(null);// Almacena la configuración del vehículo del usuario actual.
  const [cars, setCars] = useState([]);// Almacena la lista de todos los vehículos visibles en la simulación.
  const [bridgeStatus, setBridgeStatus] = useState(null);// Guarda el estado actual del puente (ocupado, dirección, colas).
  const [isLoopingStopped, setIsLoopingStopped] = useState(false);// Controla si el usuario ha detenido el ciclo de su vehículo.
  const [showStatsModal, setShowStatsModal] = useState(false);// Gestiona la visibilidad del modal de estadísticas.
  const [carStats, setCarStats] = useState(null); // Almacena las estadísticas del vehículo para mostrarlas en el modal.
  const [crossingTime, setCrossingTime] = useState(0);// Guarda el tiempo restante de cruce del vehículo del usuario.
  const [restingTime, setRestingTime] = useState(0);// Guarda el tiempo restante de descanso antes de volver a la cola.

  // Referencias para almacenar los IDs de los intervalos y temporizadores.
  const pollingIntervalRef = useRef(null);
  const heartbeatIntervalRef = useRef(null);
  const timerRef = useRef(null);

  // Efecto para gestionar la comunicación continua con el backend (polling y heartbeat).
  useEffect(() => {
    // Función de limpieza para detener todos los intervalos activos.
    const cleanupIntervals = () => {
      clearInterval(pollingIntervalRef.current);
      clearInterval(heartbeatIntervalRef.current);
      clearInterval(timerRef.current);
    };

    // Obtiene el estado completo de la simulación (colas, puente, coche actual).
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
      } catch (error) {
        console.error("Error en polling:", error);
      }
    };

    // Envía una señal 'ping' para mantener activa la sesión del vehículo en el backend.
    const sendHeartbeat = async () => {
      if (carConfig?.id) {
        try {
          const res = await fetch(`/api/vehicle/${carConfig.id}/ping`, { method: 'POST' });
          if (!res.ok) {
            console.warn("Sesión expirada. Deteniendo comunicación.");
            cleanupIntervals();
          }
        } catch (error) {
          console.error("Error en heartbeat:", error);
        }
      }
    };

    // Inicia los intervalos de polling y heartbeat si hay un coche configurado.
    if (carConfig?.id) {
      pollingIntervalRef.current = setInterval(fetchSimulationState, 1000);
      heartbeatIntervalRef.current = setInterval(sendHeartbeat, 5000);
    }

    // Función de limpieza que se ejecuta al desmontar el componente.
    return cleanupIntervals;
  }, [carConfig?.id]);

  // Efecto para gestionar los temporizadores de cuenta regresiva (cruce y descanso).
  useEffect(() => {
    // Limpia cualquier temporizador previo al re-ejecutarse.
    clearInterval(timerRef.current);
    setCrossingTime(0);
    setRestingTime(0);

    const myStatus = carConfig?.status;
    const mySpeed = carConfig?.speed;
    const canRequeueAt = carConfig?.CanRequeueAt;

    // Inicia el temporizador de cruce si el coche está cruzando.
    if (myStatus === 'crossing') {
      let timeLeft = Math.round(4 + (10 - mySpeed) / 9 * 8);
      setCrossingTime(timeLeft);
      timerRef.current = setInterval(() => {
        setCrossingTime(prev => (prev > 0 ? prev - 1 : 0));
      }, 1000);
      // Inicia el temporizador de descanso si el coche está esperando para volver a la cola.
    } else if (canRequeueAt > 0) {
      const now = Math.floor(Date.now() / 1000);
      const timeLeft = canRequeueAt - now;

      if (timeLeft > 0) {
        setRestingTime(timeLeft);
        timerRef.current = setInterval(() => {
          setRestingTime(prev => (prev > 0 ? prev - 1 : 0));
        }, 1000);
      }
    }

    // Limpia el temporizador al desmontar el componente o cambiar dependencias.
    return () => clearInterval(timerRef.current);
  }, [carConfig]);

  // Efecto de inicialización. Se ejecuta una vez para registrar el vehículo.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    let dir = params.get('dir');
    let vel = parseInt(params.get('vel'), 10);

    // Si no hay parámetros en la URL, genera valores aleatorios.
    if (!dir || !vel || isNaN(vel)) {
      dir = Math.random() < 0.5 ? 'NORTE' : 'SUR';
      vel = Math.floor(Math.random() * 10) + 1;
    }

    // Registra el vehículo con los datos obtenidos.
    handleModalSubmit({ direccion: dir, velocidad: vel });
  }, []);

  // Registra el vehículo en el servidor y opcionalmente genera más vehículos.
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

      const params = new URLSearchParams(window.location.search);
      const isFromURL = params.has('dir') && params.has('vel');

      // Si es el primer vehículo, crea vehículos adicionales en nuevas pestañas.
      if (!isFromURL) {
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

  // Envía la señal al backend para que el vehículo no vuelva a la cola.
  const handleStopLoop = async () => {
    if (!carConfig || isLoopingStopped) return;
    try {
      await fetch(`/api/vehicle/${carConfig.id}/stop`, { method: 'POST' });
      setIsLoopingStopped(true);
    } catch (error) {
      console.error("Error al detener:", error);
    }
  };

  // Solicita las estadísticas del vehículo al backend y muestra el modal.
  const handleShowStats = async () => {
    if (!carConfig) return;
    try {
      const response = await fetch(`/api/vehicle/${carConfig.id}/stats`);
      if (response.ok) {
        setCarStats(await response.json());
        setShowStatsModal(true);
      }
    } catch (error) {
      console.error("Error al obtener stats:", error);
    }
  };

  // Renderiza la interfaz de la simulación.
  return (
    <div className="container-simulacion">
      <header className="header">
        <div className="header-left">
          {/* Muestra el botón de detener o el de ver estadísticas */}
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

      {/* Renderiza el modal de estadísticas si está activo */}
      {showStatsModal && (
        <StatsModal
          stats={carStats}
          carId={carConfig?.id}
          onClose={() => setShowStatsModal(false)}
        />
      )}

      {/* Renderiza el contenido principal solo si el vehículo está configurado */}
      {carConfig && (
        <main className="main-layout">
          {/* Panel lateral que muestra la información del vehículo y del puente */}
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

          {/* Panel central que contiene la visualización gráfica de la simulación */}
          <section className="center-panel">
            <div className="simulacion-box">
              <div className="bridge-container">
                <img src="/bridge-sprite.png" alt="Puente" className="bridge-image" />
                <img src="/water-sprite.png" alt="Agua" className="water-image" />
                <div className="cars-lane">
                  {/* Renderiza cada vehículo en la simulación */}
                  {cars.map(car => (
                    <Car
                      key={car.id}
                      {...car}
                      isLocal={car.id === carConfig?.id} 
                      animationDuration={car.status === 'crossing' && car.id === carConfig?.id ? crossingTime : 0}
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