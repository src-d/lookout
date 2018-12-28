import React from 'react';

interface ErrorProps {
  errors: string[];
}

function Errors({ errors }: ErrorProps) {
  return <div>{errors.join(',')}</div>;
}

export default Errors;
