import { useState } from 'react';
import '../styles/components.css'; 

// Componente que renderiza el modal inicial para que el usuario configure la dirección y velocidad de su vehículo antes de comenzar la simulación.
export const ModalInicial = ({ onSubmit }) => {
  const [direccion, setDireccion] = useState('Norte');
  const [velocidad, setVelocidad] = useState(5);

  const handleDirectionChange = (e) => {
    setDireccion(e.target.value === '0' ? 'Norte' : 'Sur');
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit({ direccion, velocidad });
  };

  return (
    <div className="modal-container">
      <div className="modal-box">
        <form onSubmit={handleSubmit} className="modal-form">
          <h2>Configura tu Vehículo</h2>

          <div className="form-group">
            <label htmlFor="direction-slider">
              Dirección: <strong>{direccion}</strong>
            </label>
            <div className="slider-wrapper">
              <span>Norte</span>
              <input
                type="range"
                id="direction-slider"
                min="0"
                max="1"
                step="1"
                value={direccion === 'Norte' ? '0' : '1'}
                onChange={handleDirectionChange}
                className="direction-slider"
              />
              <span>Sur</span>
            </div>
          </div>

          <div className="form-group">
            <label htmlFor="velocity-input">Velocidad (1-10):</label>
            <input
              id="velocity-input"
              type="number"
              min="1"
              max="10"
              value={velocidad}
              onChange={(e) => setVelocidad(parseInt(e.target.value))}
              required
              className="velocity-input"
            />
          </div>

          <button type="submit" className="submit-button">
            Ingresar a la Simulación
          </button>
        </form>
      </div>
    </div>
  );
};