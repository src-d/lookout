interface LookoutApiOptions {
  SERVER_URL?: string;
}

declare global {
  interface Window {
    lookout: LookoutApiOptions;
  }
}

export default window.lookout || {};
