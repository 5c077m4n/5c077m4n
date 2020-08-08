const fetch = require('cross-fetch');

async function getPkgData(pkgName) {
	const npmPackageDataUrl = 'https://api.npms.io/v2/package/';
	const data = await fetch(npmPackageDataUrl + pkgName);
	const json = await data.json();

	return {
		pkgName,
		downloadCount: json.collected.npm.downloads.reduce((total, { count }) => count + total, 0),
		quality: json.score.detail.quality,
		starCount: json.collected.github?.starsCount,
	};
}

async function main() {
	const httpResponderData = await getPkgData('http-responder');
	const pkgplayData = await getPkgData('pkgplay');
	const awaitFnData = await getPkgData('await-fn');

	console.log(httpResponderData, pkgplayData, awaitFnData);
}

main();
