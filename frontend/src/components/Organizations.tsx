import React from 'react';
import * as api from '../api';
import { User } from '../services/auth';
import Errors from './Errors';
import Loader from './Loader';

interface OrgsProps {
  user: User;
}

interface OrgsState {
  done: boolean;
  orgs: api.OrgListItem[];
  errors: string[];
}

class Organizations extends React.Component<OrgsProps, OrgsState> {
  public state: OrgsState = {
    done: false,
    orgs: [],
    errors: []
  };

  public componentDidMount() {
    return api
      .orgs()
      .then(resp => {
        this.setState({
          done: true,
          orgs: resp,
          errors: []
        });
      })
      .catch(err => {
        this.setState({
          done: true,
          orgs: [],
          errors: err
        });
      });
  }

  public render() {
    if (!this.state.done) {
      return <Loader />;
    }

    if (this.state.errors.length > 0) {
      return <Errors errors={this.state.errors} />;
    }

    const orgs = this.state.orgs.map(org => (
      <li key={org.name}>
        <a href={`/org/${org.name}`}>{org.name}</a>
      </li>
    ));

    return (
      <div>
        <h1>
          Organizations with source
          {'{d}'} Lookout installed where {this.props.user.name} is admin:
        </h1>
        <ul>{orgs}</ul>
      </div>
    );
  }
}

export default Organizations;
