import * as path from 'path';
import { fileURLToPath } from 'url';

import fetch from 'cross-fetch';
import fs from 'fs-extra';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const numberFormatter = new Intl.NumberFormat();
const percentFormatter = new Intl.NumberFormat('en-GB', { style: 'percent' });

async function getPkgData(pkgName) {
	const npmPackageDataUrl = 'https://api.npms.io/v2/package/';
	const data = await fetch(npmPackageDataUrl + pkgName);
	const json = await data.json();

	return {
		pkgName,
		downloadCount: json.collected.npm.downloads.reduce((total, { count }) => count + total, 0),
		quality: json.score.detail.quality,
		coverage: json.collected.source.coverage,
	};
}

async function main() {
	const packagesData = await Promise.all([
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
				numberFormatter.format(packagesData.reduce((total, { downloadCount }) => total + downloadCount, 0)),
			)
			.replace(
				/\{\{\s*avgQuality\s*\}\}/g,
				percentFormatter.format(
					packagesData.reduce((total, { quality }) => total + quality, 0) / packagesData.length,
				),
			)
			.replace(
				/\{\{\s*codeCov\s*\}\}/g,
				percentFormatter.format(
					packagesData.reduce((total, { coverage }) => total + (coverage ?? 0), 0) /
						packagesData.filter((pkg) => pkg.coverage).length,
				),
			)
			.replace(/\{\{\s*dateNow\s*\}\}/g, new Date().toLocaleDateString()),
	);
}

main();
