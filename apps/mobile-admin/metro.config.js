const { getDefaultConfig } = require('expo/metro-config');
const { withNativeWind } = require('nativewind/metro');
const { FileStore } = require('metro-cache');
const path = require('path');

const projectRoot = __dirname;

const config = getDefaultConfig(projectRoot);

config.cacheStores = [
  new FileStore({ root: path.join(projectRoot, '.metro-cache') }),
];

module.exports = withNativeWind(config, {
  input: './global.css',
});
