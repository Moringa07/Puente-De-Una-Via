// Componente que renderiza visualmente un único vehículo en la simulación.
// Aplica clases CSS y estilos en línea para controlar su apariencia, posición y la animación de cruce, basándose en las propiedades recibidas.
export const Car = ({ id, direction, status, spriteType, animationDuration }) => {
  const carClassName = `car-sprite direction-${direction.toLowerCase()} ${status === 'crossing' ? 'crossing' : ''}`;

  const carStyle = status === 'crossing'
    ? { animationDuration: `${animationDuration}s` }
    : {};

  return (
    <img
      src={`/car${spriteType}.png`}
      alt={`Auto ${id}`}
      className={carClassName}
      style={carStyle}
    />
  );
};