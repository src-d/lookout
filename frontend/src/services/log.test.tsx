import log from './log';

it('log service do nothing in testing env', () => {
  expect(log).toHaveProperty('error');

  const logErrorCode = log.error
    .toString()
    .replace(/\s*/g, '')
    .trim();
  const noopCode = '()=>undefined';
  expect(logErrorCode).toEqual(noopCode);
});
