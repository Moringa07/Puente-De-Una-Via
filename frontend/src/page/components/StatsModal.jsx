import React from 'react';
import '../styles/components.css';

// Componente que muestra un modal con las estadísticas de rendimiento finales del vehículo del usuario, una vez que la simulación ha terminado para él.
export const StatsModal = ({ stats, carId, onClose }) => {
  return (
    <div className="modal-container">
      <div className="modal-box stats-modal">
        <h2>Estadísticas del Vehículo</h2>
        <p className="stats-id">ID: <strong>{carId}</strong></p>

        <div className="stats-grid">
          <div className="stat-item">
            <span className="stat-label">Cruces Totales</span>
            <span className="stat-value">{stats?.total_crossings ?? '...'}</span>
          </div>

          <div className="stat-item">
            <span className="stat-label">Tiempo Total en Puente</span>
            <span className="stat-value">
              {stats?.total_time_on_bridge_sec?.toFixed(2) ?? '...'} s
            </span>
          </div>

          <div className="stat-item">
            <span className="stat-label">Promedio por Cruce</span>
            <span className="stat-value">
              {stats?.avg_crossing_time_sec?.toFixed(2) ?? '...'} s
            </span>
          </div>

          <div className="stat-item">
            <span className="stat-label">Tiempo Total Esperando</span>
            <span className="stat-value">
              {stats?.total_waiting_time_sec?.toFixed(2) ?? '...'} s
            </span>
          </div>

          <div className="stat-item">
            <span className="stat-label">Promedio Espera</span>
            <span className="stat-value">
              {stats?.avg_waiting_time_sec?.toFixed(2) ?? '...'} s
            </span>
          </div>

          <div className="stat-item">
            <span className="stat-label">% Tiempo en Puente</span>
            <span className="stat-value">
              {stats?.time_in_bridge_percent?.toFixed(2) ?? '...'}%
            </span>
          </div>
        </div>

        <button onClick={onClose} className="submit-button">
          Cerrar
        </button>
      </div>
    </div>
  );
};