import React from 'react';
import * as api from '../api';
import { User } from '../services/auth';
import Errors from './Errors';
import Loader from './Loader';

interface OrgProps {
  user: User;
  orgName: string;
}

interface OrgState {
  done: boolean;
  org: api.OrgResponse | undefined;
  errors: string[];

  config: string;
}

class Organization extends React.Component<OrgProps, OrgState> {
  constructor(props: OrgProps) {
    super(props);

    this.state = {
      done: false,
      org: undefined,
      errors: [],
      config: ''
    };

    this.handleConfigChange = this.handleConfigChange.bind(this);
    this.handleConfigSave = this.handleConfigSave.bind(this);
  }

  public componentDidMount() {
    return api
      .org(this.props.orgName)
      .then(resp =>
        this.setState({
          done: true,
          org: resp,
          errors: [],
          config: resp.config
        })
      )
      .catch(err => {
        this.setState({
          done: true,
          org: undefined,
          errors: err,
          config: ''
        });
      });
  }

  public render() {
    if (!this.state.done) {
      return <Loader />;
    }

    const org = this.state.org;

    if (this.state.errors.length > 0 || org === undefined) {
      return <Errors errors={this.state.errors} />;
    }

    return (
      <div>
        <h1>Settings for Organization {org.name}</h1>
        <textarea
          value={this.state.config}
          onChange={this.handleConfigChange}
          style={{ font: 'monospace', width: '300px', height: '10em' }}
        />
        <div>
          <br />
          <button onClick={this.handleConfigSave}>Save</button>
        </div>
      </div>
    );
  }

  private handleConfigChange(event: React.ChangeEvent<HTMLTextAreaElement>) {
    this.setState({ config: event.target.value });
  }

  private handleConfigSave() {
    if (this.state.org === undefined) {
      // tslint:disable-next-line
      console.error('handleConfigSave called with undefined state.org');
      return;
    }

    api
      .updateConfig(this.state.org.name, this.state.config)
      .then(resp =>
        this.setState({
          done: true,
          org: resp,
          errors: [],
          config: resp.config
        })
      )
      .catch(err => {
        this.setState({
          done: true,
          org: undefined,
          errors: err,
          config: ''
        });
      });
  }
}

export default Organization;
