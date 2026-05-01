const path = require('path');
const { getDefaultConfig } = require('expo/metro-config');
const { withNativeWind } = require('nativewind/metro');

const config = getDefaultConfig(__dirname);
const repoRoot = path.resolve(__dirname, '../..');

config.watchFolders = [repoRoot];
config.resolver.nodeModulesPaths = [
  path.resolve(__dirname, 'node_modules'),
  path.resolve(repoRoot, 'node_modules'),
];

module.exports = withNativeWind(config, { input: './global.css' });
