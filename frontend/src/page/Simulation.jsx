import { useState } from 'react';
import './styles/simulation.css';
import { ModalInicial } from './components/ModalInicial';
import { Car } from './components/car';

const getRandomSpriteType = () => Math.floor(Math.random() * 4) + 1;

export default function Simulation() {
  const [showModal, setShowModal] = useState(true);
  const [carConfig, setCarConfig] = useState(null);
  
  const [cars, setCars] = useState([]);

  const handleModalSubmit = ({ direccion, velocidad }) => {
    const userCarSprite = getRandomSpriteType();
    
    setCarConfig({
      id: `auto_${Date.now()}`,
      direccion,
      velocidad,
      spriteType: userCarSprite 
    });
    
    setShowModal(false);
  };

  const handleAddNewCar = () => {
    const newCar = {
      id: `auto_${Date.now()}`,
      direction: Math.random() > 0.5 ? 'Norte' : 'Sur', 
      spriteType: getRandomSpriteType(), 
      position: { top: '42%', left: '-10%' } 
    };

    setCars(prevCars => [...prevCars, newCar]);
  };

  return (
    <div className="container-simulacion">
      <header className="header"><h1>Puente de Una Vía</h1><p>Simulación de Sistema Distribuido</p></header>

      {showModal && <ModalInicial onSubmit={handleModalSubmit} />}

      {carConfig && !showModal && (
        <main className="main-layout">
          <aside className="left-panel">
            <div className="panel-box">
              <h3>Tu Vehículo</h3>
              <p><strong>ID:</strong> {carConfig.id}</p>
              <div className="placeholder-sprite">
                <img 
                  src={`/car${carConfig.spriteType}.png`}
                  alt="Icono de tu auto"
                  className="car-icon-sprite" 
                />
              </div>
            </div>

            <div className="panel-box"><h3>Estado del Vehículo</h3><p><strong>Dirección:</strong> {carConfig.direccion}</p><p><strong>Velocidad:</strong> {carConfig.velocidad}</p><p><strong>Estado:</strong> <span className="status-waiting">En espera</span></p></div>
            <div className="panel-box"><h3>Estado del Puente</h3><p><strong>Dirección Actual:</strong> Norte</p><p><strong>Vehículos Cruzando:</strong> {cars.length}</p><p><strong>Semáforo:</strong> <span className="status-go">VERDE</span></p></div>
            
            <button onClick={handleAddNewCar} className="add-car-button">
              Simular Conexión de Otro Auto
            </button>
          </aside>

          <section className="center-panel">
            <div className="simulacion-box">
              <div className="bridge-container">
                <img 
                  src="/bridge-sprite.png" 
                  alt="Puente de una vía" 
                  className="bridge-image" 
                />
                <div className="cars-lane">
                  {cars.map(car => (
                    <Car
                      key={car.id}
                      id={car.id}
                      direction={car.direction}
                      position={car.position}
                      spriteType={car.spriteType} 
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