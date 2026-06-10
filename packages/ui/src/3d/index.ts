import { Application } from "pixi.js";

export { CatScene } from "./CatScene";
export type { CatSceneProps } from "./CatScene";
export { LowPolyCat } from "./LowPolyCat";
export type { LowPolyCatProps } from "./LowPolyCat";
export { R3FScene } from "./Scene";
export type { R3FSceneProps } from "./Scene";

export async function createPixiApp(canvas: HTMLCanvasElement): Promise<Application> {
  const app = new Application();
  await app.init({ canvas, background: "#FFF7F0", antialias: true });
  return app;
}
