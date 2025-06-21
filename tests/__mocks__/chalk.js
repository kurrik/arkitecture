// Mock for chalk module to handle ES module issues in Jest
const chalk = {
  red: (text) => text,
  green: (text) => text,
  blue: (text) => text,
  yellow: (text) => text,
  gray: (text) => text,
  grey: (text) => text,
  magenta: (text) => text,
  cyan: (text) => text,
  white: (text) => text,
  black: (text) => text,
};

module.exports = chalk;