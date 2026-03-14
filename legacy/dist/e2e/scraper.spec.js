"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const test_1 = require("@playwright/test");
const fs_1 = __importDefault(require("fs"));
const fast_xml_parser_1 = require("fast-xml-parser");
const urlPrefix = "https://www.perseus.tufts.edu/hopper";
const firstLetterOfLexicon = "text?doc=Perseus%3Atext%3A1999.03.0079";
(0, test_1.test)("scrape", async ({ page }) => {
    var _a, _b, _c, _d, _e, _f;
    let previousLetter = null;
    let currentLetter = firstLetterOfLexicon;
    let nextLetter = null;
    test_1.test.setTimeout(620000);
    await page.goto(`${urlPrefix}/${currentLetter}`);
    let currentCount = 0;
    const maxRecords = 50;
    const returnValues = [];
    while (currentCount <= maxRecords) {
        const xmlPath = await page
            .locator("#footer_nav")
            .getByRole("link", { name: "view as XML" })
            .getAttribute("href");
        nextLetter = await page
            .locator("#footer_nav")
            .getByRole("link", { name: "next" })
            .getAttribute("href");
        const xmlFile = await page.goto(`${urlPrefix}/${xmlPath}`);
        const xmlBody = await (xmlFile === null || xmlFile === void 0 ? void 0 : xmlFile.body());
        currentCount++;
        if (xmlBody) {
            const parser = new fast_xml_parser_1.XMLParser();
            let parsedXml = parser.parse(xmlBody);
            const returnValue = {
                entry: Object.assign({}, parsedXml),
                previousLetter,
                currentLetter,
                nextLetter,
            };
            returnValues.push(returnValue);
        }
        else {
            console.warn("no xml body detected");
        }
        await page.goBack();
        await page
            .locator("#footer_nav")
            .getByRole("link", { name: "next" })
            .click();
        previousLetter = currentLetter;
        currentLetter = nextLetter;
    }
    const limitedValues = returnValues.slice(0, 200);
    const firstWord = ((_c = (_b = (_a = limitedValues[0]) === null || _a === void 0 ? void 0 : _a.entry) === null || _b === void 0 ? void 0 : _b.headword) === null || _c === void 0 ? void 0 : _c[0]) || "unknown";
    const lastWord = ((_f = (_e = (_d = limitedValues[limitedValues.length - 1]) === null || _d === void 0 ? void 0 : _d.entry) === null || _e === void 0 ? void 0 : _e.headword) === null || _f === void 0 ? void 0 : _f[0]) || "unknown";
    const filename = `${firstWord}-${lastWord}.json`;
    fs_1.default.writeFileSync(filename, JSON.stringify(limitedValues, null, 2));
});
//# sourceMappingURL=scraper.spec.js.map