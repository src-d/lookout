import * as api from '../api';

export interface User {
  name: string;
}

const localStorageKey = 'token';

class AuthService {
  private _user: User | null = null;

  get token() {
    return window.localStorage.getItem(localStorageKey);
  }

  get isAuthenticated() {
    if (!window.localStorage.getItem(localStorageKey)) {
      return Promise.resolve(false);
    }

    if (!this._user) {
      return api
        .me()
        .then(resp => {
          this._user = resp;
          return true;
        })
        .catch(err => {
          // tslint:disable-next-line
          console.error(err);
          this._user = null;
          return false;
        });
    }

    return Promise.resolve(true);
  }

  get user() {
    return this._user;
  }

  get loginUrl() {
    return api.loginUrl;
  }

  public callback(queryString: string) {
    return api.callback(location.search).then(resp => {
      window.localStorage.setItem(localStorageKey, resp.token);
    });
  }

  public logout(): void {
    window.localStorage.removeItem(localStorageKey);
  }
}

export default new AuthService();
