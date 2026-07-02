import { describe, expect, it } from "vitest";
import { escapeCsvCell, formatCsv } from "../format-csv";

describe("formatCsv", () => {
  it("quotes cells with commas and newlines", () => {
    expect(escapeCsvCell('say "hello"')).toBe('"say ""hello"""');
    expect(formatCsv(["City"], [["New York, NY"]])).toBe('City\n"New York, NY"');
  });

  it("joins headers and rows", () => {
    const csv = formatCsv(
      ["a", "b"],
      [
        ["1", "2"],
        ["x", "y"],
      ],
    );
    expect(csv).toBe("a,b\n1,2\nx,y");
  });
});
