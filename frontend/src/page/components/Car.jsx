import React from 'react';

export const Car = ({ id, direction, position, spriteType }) => {
  const carClassName = `car-sprite ${direction === 'Sur' ? 'flipped' : ''}`;

  const carImageSrc = `/car${spriteType}.png`;

  return (
    <img
      src={carImageSrc}
      alt={`Auto ${id}`}
      className={carClassName}
      style={{
        top: position.top,
        left: position.left,
      }}
    />
  );
};