import Auth from './services/auth';
import lookoutOptions from './services/options';

export const serverUrl = lookoutOptions.SERVER_URL || 'http://127.0.0.1:8080';

const apiUrl = (url: string) => `${serverUrl}${url}`;

interface ApiCallOptions {
  method?: string;
  body?: object;
}

interface ServerError {
  title: string;
  description: string;
}

export function apiCall<T>(
  url: string,
  options: ApiCallOptions = {}
): Promise<T> {
  const fetchOptions: RequestInit = {
    ...options,
    credentials: 'include',
    headers: {
      Authorization: `Bearer ${Auth.token}`,
      'Content-Type': 'application/json'
    },
    body: null
  };

  if (options.body) {
    fetchOptions.body = JSON.stringify(options.body);
  }

  return fetch(apiUrl(url), fetchOptions).then(response => {
    if (!response.ok) {
      // when server return Unauthorized we need to remove token
      if (response.status === 401) {
        Auth.logout();
      }

      return response
        .json()
        .catch(() => {
          throw [response.statusText];
        })
        .then(json => {
          let errors: string[];

          try {
            errors = (json as { errors: ServerError[] }).errors.map(
              e => e.title
            );
          } catch (e) {
            errors = [e.toString()];
          }

          throw errors;
        });
    }

    return response.json().then(json => (json as { data: T }).data);
  });
}

export const loginUrl = apiUrl('/login');

interface AuthResponse {
  token: string;
}

export function callback(queryString: string): Promise<AuthResponse> {
  return apiCall<AuthResponse>(`/api/callback${queryString}`);
}

interface MeResponse {
  name: string;
}

export function me(): Promise<MeResponse> {
  return apiCall<MeResponse>('/api/me');
}

export interface OrgListItem {
  id: number;
  name: string;
}

interface OrgsResponse extends Array<OrgListItem> {}

// Returns a list of GitHub organization names where lookout is installed and
// the logged-in user is an administrator
export function orgs(): Promise<OrgsResponse> {
  return apiCall<OrgsResponse>('/api/orgs');
}

export interface OrgResponse {
  id: number;
  name: string;
  config: string;
}

// Returns info about the individual organization
export function org(name: string): Promise<OrgResponse> {
  return apiCall<OrgResponse>(`/api/org/${name}`);
}

// Updates the organization config
export function updateConfig(
  name: string,
  config: string
): Promise<OrgResponse> {
  return apiCall<OrgResponse>(`/api/org/${name}`, {
    method: 'PUT',
    body: { config }
  });
}
