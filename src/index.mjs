import { readFile, writeFile } from "node:fs/promises";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

const numberFormatter = new Intl.NumberFormat();
const percentFormatter = new Intl.NumberFormat("en-GB", { style: "percent" });
const placeholderRegex = str => new RegExp(String.raw`\{\{\s*${str}\s*\}\}`, "g");

async function getPkgData(pkgName, npmPackageDataUrl = "https://api.npms.io/v2/package/") {
	const data = await globalThis.fetch(npmPackageDataUrl + pkgName);
	const json = await data.json();

	return {
		pkgName,
		downloadCount: json.collected.npm.downloads.reduce((total, { count }) => total + count, 0),
		quality: json.score.detail.quality,
		coverage: json.collected.source.coverage,
	};
}

async function main() {
	const packagesData = await Promise.all([
		getPkgData("http-responder"),
		getPkgData("pkgplay"),
		getPkgData("await-fn"),
	]);

	const readmeTemplate = await readFile(resolve(__dirname, "./readme-template.md"), "utf8");
	await writeFile(
		resolve(__dirname, "../README.md"),
		readmeTemplate
			.replace(
				placeholderRegex("downloadsCount"),
				numberFormatter.format(
					packagesData
						.filter(pkg => pkg.downloadCount)
						.reduce((total, { downloadCount }) => total + downloadCount, 0),
				),
			)
			.replace(
				placeholderRegex("avgQuality"),
				percentFormatter.format(
					packagesData
						.filter(pkg => pkg.quality)
						.reduce((total, { quality }, _, origin) => total + quality / origin.length, 0),
				),
			)
			.replace(
				placeholderRegex("codeCov"),
				percentFormatter.format(
					packagesData
						.filter(pkg => pkg.coverage)
						.reduce((total, { coverage }, _, origin) => total + coverage / origin.length, 0),
				),
			)
			.replace(placeholderRegex("todayDate"), new Date().toDateString()),
	);
}

main()
	.then(() => console.log("Generated readme file successfully."))
	.catch(error => {
		console.error(error);
		process.exit(1);
	});
