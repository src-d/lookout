import React, { Component } from 'react';
import {
  BrowserRouter as Router,
  Link,
  Redirect,
  Route,
  RouteComponentProps,
  RouteProps
} from 'react-router-dom';
import './App.css';
import Callback from './Callback';
import Loader from './components/Loader';
import Organizations from './components/Organizations';
import Auth, { User } from './services/auth';

function Login() {
  return (
    <header className="App-header">
      <a className="App-link" href={Auth.loginUrl}>
        Login using Github
      </a>
    </header>
  );
}

function Logout() {
  Auth.logout();

  return <Redirect to="/" />;
}

interface IndexProps {
  user: User;
}

function Index({ user }: IndexProps) {
  return (
    <div>
      <header className="App-header">
        Hello {user.name}! <Link to="/logout">Logout</Link>
      </header>
      <Organizations user={user} />
    </div>
  );
}

interface PrivateRouteState {
  isAuthenticated: boolean | undefined;
}

interface PrivateRouteComponentProps<P> extends RouteComponentProps {
  user: User | null;
}

interface PrivateRouteProps extends RouteProps {
  component:
    | React.ComponentType<PrivateRouteComponentProps<any>>
    | React.ComponentType<any>;
}

function PrivateRoute({ component, ...rest }: PrivateRouteProps) {
  class CheckAuthComponent extends Component<
    RouteComponentProps,
    PrivateRouteState
  > {
    constructor(props: RouteComponentProps) {
      super(props);

      this.state = { isAuthenticated: undefined };
    }

    public componentDidMount() {
      Auth.isAuthenticated
        .then(ok => this.setState({ isAuthenticated: ok }))
        .catch(() => this.setState({ isAuthenticated: false }));
    }

    public render() {
      if (!component) {
        return null;
      }

      if (this.state.isAuthenticated === true) {
        // tslint:disable-next-line
        const WrappedComponent = component; // must be uppercase because of JSX
        return <WrappedComponent {...this.props} user={Auth.user} />;
      }

      if (this.state.isAuthenticated === false) {
        return (
          <Redirect
            to={{
              pathname: '/login',
              state: { from: this.props.location }
            }}
          />
        );
      }

      return <Loader />;
    }
  }

  return <Route {...rest} component={CheckAuthComponent} />;
}

function AppRouter() {
  return (
    <Router>
      <div className="App">
        <PrivateRoute path="/" exact={true} component={Index} />
        <Route path="/login" component={Login} />
        <Route path="/logout" component={Logout} />
        <Route path="/callback" component={Callback} />
      </div>
    </Router>
  );
}

export default AppRouter;
