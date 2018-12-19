import React, { Component, ReactElement } from 'react';
import Token from './services/token';
import * as api from './api';
import './App.css';

function Loader() {
  return <div>loading...</div>;
}

interface ErrorProps {
  errors: string[];
}

function Errors({ errors }: ErrorProps) {
  return <div>{errors.join(',')}</div>;
}

function Login() {
  return (
    <header className="App-header">
      <a className="App-link" href={api.loginUrl}>
        Login using Github
      </a>
    </header>
  );
}

interface HelloProps {
  name: string;
}

function Hello({ name }: HelloProps) {
  return <header className="App-header">Hello {name}!</header>;
}

interface AppState {
  // we need undefined state for initial render
  loggedIn: boolean | undefined;
  name: string;
  errors: string[];
}

class App extends Component<{}, AppState> {
  constructor(props: {}) {
    super(props);

    this.fetchState = this.fetchState.bind(this);

    this.state = {
      loggedIn: undefined,
      name: '',
      errors: []
    };
  }

  componentDidMount() {
    // TODO: add router and use it instead of this "if"
    if (window.location.pathname === '/callback') {
      api
        .callback(window.location.search)
        .then(resp => {
          Token.set(resp.token);
          window.history.replaceState({}, '', '/');
        })
        .then(this.fetchState)
        .catch(errors => this.setState({ errors }));
      return;
    }

    if (!Token.exists()) {
      this.setState({ loggedIn: false });
      return;
    }

    // ignore error here, just ask user to re-login
    // it would cover all cases like expired token, changes on backend and so on
    this.fetchState().catch(err => console.error(err));
  }

  fetchState() {
    return api
      .me()
      .then(resp => this.setState({ loggedIn: true, name: resp.name }))
      .catch(err => {
        this.setState({ loggedIn: false });

        throw err;
      });
  }

  render() {
    const { loggedIn, name, errors } = this.state;

    let content: ReactElement<any>;
    if (errors.length) {
      content = <Errors errors={errors} />;
    } else if (typeof loggedIn === 'undefined') {
      content = <Loader />;
    } else {
      content = loggedIn ? <Hello name={name} /> : <Login />;
    }

    return <div className="App">{content}</div>;
  }
}

export default App;
