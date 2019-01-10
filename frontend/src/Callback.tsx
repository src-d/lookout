import * as H from 'history';
import React, { Component } from 'react';
import { Redirect } from 'react-router-dom';
import * as api from './api';
import Errors from './components/Errors';
import Loader from './components/Loader';
import Auth from './services/auth';

interface CallbackProps {
  location: H.Location;
}

interface CallbackState {
  success: boolean;
  errors: string[];
}

class Callback extends Component<CallbackProps, CallbackState> {
  constructor(props: CallbackProps) {
    super(props);

    this.state = {
      success: false,
      errors: []
    };
  }

  public componentDidMount() {
    Auth.callback(this.props.location.search)
      .then(() => this.setState({ success: true }))
      .catch(errors => this.setState({ errors }));
  }

  public render() {
    const { errors, success } = this.state;

    if (errors.length) {
      return <Errors errors={errors} />;
    }

    if (success) {
      const { from } = this.props.location.state || { from: { pathname: '/' } };
      return <Redirect to={from} />;
    }

    return <Loader />;
  }
}

export default Callback;
