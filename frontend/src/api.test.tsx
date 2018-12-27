import { GlobalWithFetchMock } from 'jest-fetch-mock';
import Token from './services/token';
import { apiCall } from './api';

// can be moved to setupFiles later if needed
const customGlobal: GlobalWithFetchMock = global as GlobalWithFetchMock;
customGlobal.fetch = require('jest-fetch-mock');
customGlobal.fetchMock = customGlobal.fetch;

describe('api', () => {
  beforeEach(() => {
    fetchMock.resetMocks();
  });

  it('apiCall ok', () => {
    Token.set('token');
    fetchMock.mockResponseOnce(JSON.stringify({ data: 'result' }));

    return apiCall('/test').then(resp => {
      expect(resp).toEqual('result');

      const call = fetchMock.mock.calls[0];
      const [url, opts] = call;
      expect(url).toEqual('http://127.0.0.1:8080/test');
      expect(opts.headers.Authorization).toEqual('Bearer token');
    });
  });

  it('apiCall http error', () => {
    fetchMock.mockResponseOnce('', { status: 500 });

    return apiCall('/test').catch(err => {
      expect(err).toEqual(['Internal Server Error']);
    });
  });

  it('apiCall http error with custom text', () => {
    fetchMock.mockResponseOnce('', { status: 404, statusText: 'Custom text' });

    return apiCall('/test').catch(err => {
      expect(err).toEqual(['Custom text']);
    });
  });

  it('apiCall http error with json response', () => {
    fetchMock.mockResponseOnce(
      JSON.stringify({ errors: [{ title: 'err1' }, { title: 'err2' }] }),
      {
        status: 500
      }
    );

    return apiCall('/test').catch(err => {
      expect(err).toEqual(['err1', 'err2']);
    });
  });

  it('apiCall removes token on unauthorized response', () => {
    Token.set('token');
    fetchMock.mockResponseOnce('', { status: 401 });

    return apiCall('/test').catch(err => {
      expect(err).toEqual(['Unauthorized']);
      expect(Token.get()).toBe(null);
    });
  });
});
