import { describe, expect, it } from "vitest"
import { validateSAJson } from "./sa-validate"

describe("validateSAJson", () => {
  it("empty → saInvalidJson", () => {
    expect(validateSAJson("")).toEqual({ ok: false, errorKey: "saInvalidJson" })
    expect(validateSAJson("   ")).toEqual({ ok: false, errorKey: "saInvalidJson" })
  })

  it("malformed JSON → saInvalidJson", () => {
    expect(validateSAJson("{not-json")).toEqual({ ok: false, errorKey: "saInvalidJson" })
  })

  it("wrong type → saNotServiceAccount", () => {
    expect(validateSAJson(JSON.stringify({ type: "user_account" }))).toEqual({
      ok: false,
      errorKey: "saNotServiceAccount",
    })
  })

  it("missing required fields → saMissingFields", () => {
    expect(
      validateSAJson(JSON.stringify({ type: "service_account", project_id: "p" }))
    ).toEqual({ ok: false, errorKey: "saMissingFields" })
  })

  it("valid → ok with clientEmail + projectID", () => {
    const text = JSON.stringify({
      type: "service_account",
      project_id: "lead-cat-12345",
      private_key: "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
      client_email: "lead-cat@lead-cat-12345.iam.gserviceaccount.com",
      token_uri: "https://oauth2.googleapis.com/token",
    })
    expect(validateSAJson(text)).toEqual({
      ok: true,
      clientEmail: "lead-cat@lead-cat-12345.iam.gserviceaccount.com",
      projectID: "lead-cat-12345",
    })
  })
})
