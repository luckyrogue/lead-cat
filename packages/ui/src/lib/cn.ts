import { twMerge } from "tailwind-merge";

export type ClassValue = string | number | null | false | undefined | ClassValue[];

function flatten(values: ClassValue[]): string[] {
  const out: string[] = [];
  for (const value of values) {
    if (!value) continue;
    if (Array.isArray(value)) {
      out.push(...flatten(value));
    } else {
      out.push(String(value));
    }
  }
  return out;
}

export function cn(...inputs: ClassValue[]): string {
  return twMerge(flatten(inputs).join(" "));
}
