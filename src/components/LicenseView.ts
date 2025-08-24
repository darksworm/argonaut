import { ALL_LICENSES } from "../data/licenses.js";
import { showInkPager } from "./InkPager";

export type LicenseSessionOptions = {
  title?: string;
};

// Shows license content in the reusable ink pager
export async function runLicenseSession(
  opts: LicenseSessionOptions = {},
): Promise<void> {
  await showInkPager(ALL_LICENSES, {
    title: opts.title || "Third Party Licenses",
    searchEnabled: true,
  });
}
