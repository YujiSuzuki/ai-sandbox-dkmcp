module.exports = {
  testEnvironment: 'node',
  collectCoverageFrom: [
    'src/**/*.js',
    '!src/server.js'
  ],
  coverageDirectory: '/tmp/jest-coverage',
  coverageReporters: ['text', 'text-summary'],
  testMatch: ['**/__tests__/**/*.test.js']
};
