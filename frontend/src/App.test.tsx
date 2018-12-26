import React from 'react';
import ReactDOM from 'react-dom';
import App from './App';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(<App />, div);

  // Unmount raises error about update of unmounted component
  // due to using promises for checking authorization
  setTimeout(() => ReactDOM.unmountComponentAtNode(div), 0);
});
