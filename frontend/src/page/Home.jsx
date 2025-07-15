import React from 'react';
import { Link } from 'react-router-dom';
import './styles/home.css'
export const Home = () => {
  return (
    <div className="home-container">
      <div className="home-content-box">
        <h1 className="home-title">
          Simulación de Vehículos en un Puente de Una Vía
        </h1>
        <p className="home-description">
          Una simulación de un sistema distribuido donde múltiples vehículos
          intentan cruzar un puente de un solo carril, gestionado por un proceso de servidor central.
        </p>
        <p className="home-authors">
          Creado por: <strong>Merry-am Blanco</strong> y <strong>Mariana Mora</strong>
        </p>
        <Link to="/simulacion" className="start-button">
          Iniciar Simulación
        </Link>
      </div>
    </div>
  );
};

export default Home;