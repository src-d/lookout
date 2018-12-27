const localStorageKey = 'token';

class TokenService {
  get() {
    return window.localStorage.getItem(localStorageKey);
  }

  set(token: string) {
    return window.localStorage.setItem(localStorageKey, token);
  }

  remove() {
    return window.localStorage.removeItem(localStorageKey);
  }

  exists() {
    return !!window.localStorage.getItem(localStorageKey);
  }
}

export default new TokenService();
