const noop = () => undefined;

export default {
  // tslint:disable-next-line no-console
  error: process.env.NODE_ENV !== 'test' ? console.error.bind(console) : noop
};
