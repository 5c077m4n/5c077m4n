import * as path from 'path';
import { fileURLToPath } from 'url';

import fetch from 'cross-fetch';
import fs from 'fs-extra';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const numberFormatter = new Intl.NumberFormat();
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
	const [httpResponderData, pkgplayData, awaitFnData] = await Promise.all([
		getPkgData('http-responder'),
		getPkgData('pkgplay'),
		getPkgData('await-fn'),
	]);

	const readmeTemplateBuffer = await fs.readFile(path.resolve(__dirname, './readme-template.md'));
	const readmeTemplate = readmeTemplateBuffer.toString();
	await fs.writeFile(
		path.resolve(__dirname, '../README.md'),
		readmeTemplate
			.replace(
				/\{\{\s*downloadsCount\s*\}\}/g,
				numberFormatter.format(
					httpResponderData.downloadCount + pkgplayData.downloadCount + awaitFnData.downloadCount,
				),
			)
			.replace(
				/\{\{\s*avgQuality\s*\}\}/g,
				(((httpResponderData.quality + pkgplayData.quality + awaitFnData.quality) / 3) * 100).toFixed(1),
				+'%',
			)
			.replace(/\{\{\s*dateNow\s*\}\}/g, `Updated on ${new Date().toLocaleDateString()}`),
	);
}

main();
