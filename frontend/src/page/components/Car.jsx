// Componente que renderiza visualmente un único vehículo en la simulación.
// Aplica clases CSS y estilos en línea para controlar su apariencia, posición y la animación de cruce, basándose en las propiedades recibidas.
export const Car = ({ id, direction, status, spriteType, animationDuration, isLocal }) => {
  if (status !== 'crossing') return null;
  const carClassName = `car-sprite direction-${direction.toLowerCase()} ${status === 'crossing' && isLocal ? 'crossing' : ''}`;

  const carStyle = status === 'crossing' && isLocal
    ? { animationDuration: `${animationDuration}s` }
    : {};

   if (!isLocal) return null;

  return (
    <img
      src={`/car${spriteType}.png`}
      alt={`Auto ${id}`}
      className={carClassName}
      style={carStyle}
    />
  );
};
