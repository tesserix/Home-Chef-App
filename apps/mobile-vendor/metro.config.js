const { getDefaultConfig } = require('expo/metro-config');
const { withNativeWind } = require('nativewind/metro');
const { FileStore } = require('metro-cache');
const path = require('path');

const projectRoot = __dirname;

const config = getDefaultConfig(projectRoot);

config.cacheStores = [
  new FileStore({ root: path.join(projectRoot, '.metro-cache') }),
];


// Force one React: hoisted deps (react-native itself) must not resolve the root copy.
const singletons = ['react', 'react-dom', 'scheduler'];
const defaultResolveRequest = config.resolver.resolveRequest;
config.resolver.resolveRequest = (context, moduleName, platform) => {
  const hit = singletons.some((s) => moduleName === s || moduleName.startsWith(`${s}/`));
  if (hit) {
    return context.resolveRequest(
      { ...context, originModulePath: path.join(projectRoot, 'package.json') },
      moduleName,
      platform,
    );
  }
  return (defaultResolveRequest ?? context.resolveRequest)(context, moduleName, platform);
};

module.exports = withNativeWind(config, {
  input: './global.css',
});
