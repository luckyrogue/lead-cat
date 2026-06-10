import { Application } from "pixi.js";

export { R3FScene } from "./Scene";
export type { R3FSceneProps } from "./Scene";

export async function createPixiApp(canvas: HTMLCanvasElement): Promise<Application> {
  const app = new Application();
  await app.init({ canvas, background: "#fdf6ff", antialias: true });
  return app;
}
