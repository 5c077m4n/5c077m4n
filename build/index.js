import * as path from 'path';

import fetch from 'cross-fetch';
import fs from 'fs-extra';

async function getPkgData(pkgName) {
	const npmPackageDataUrl = 'https://api.npms.io/v2/package/';
	const data = await fetch(npmPackageDataUrl + pkgName);
	const json = await data.json();

	return {
		pkgName,
		downloadCount: json.collected.npm.downloads.reduce((total, { count }) => count + total, 0),
		quality: json.score.detail.quality,
	};
}

async function main() {
	const readmePath = path.resolve(__dirname, '../README.md');

	const [httpResponderData, pkgplayData, awaitFnData] = await Promise.all([
		getPkgData('http-responder'),
		getPkgData('pkgplay'),
		getPkgData('await-fn'),
	]);

	const currentReadme = await fs.readFile(readmePath);
	await fs.writeFile(
		readmePath,
		currentReadme
			.replace(
				/\{\{\s*downloadsCount\s*\}\}/g,
				httpResponderData.downloadCount + pkgplayData.downloadCount + awaitFnData.downloadCount,
			)
			.replace(
				/\{\{\s*avgQuality\s*\}\}/g,
				(httpResponderData.quality + pkgplayData.quality + awaitFnData.quality) / 3,
			),
	);
}

main();
