import { Suspense } from "react"
import { Canvas } from "@react-three/fiber"
import { Float, OrbitControls, useGLTF } from "@react-three/drei"

import { CatModelBoundary } from "./cat-model-boundary"
import { GltfCat } from "./gltf-cat"
import { LowPolyCat } from "./low-poly-cat"
import { StagePlatform } from "./stage-platform"

export const LEAD_CAT_MODEL_URL = "/models/lead-cat.glb"

if (typeof window !== "undefined") {
  useGLTF.preload(LEAD_CAT_MODEL_URL)
}

export interface CatSceneProps {
  className?: string
  autoRotate?: boolean
  followPointer?: boolean
  modelUrl?: string
}

function CatModel({
  followPointer,
  modelUrl,
}: {
  followPointer: boolean
  modelUrl: string
}) {
  return (
    <CatModelBoundary fallback={<LowPolyCat followPointer={followPointer} />}>
      <GltfCat src={modelUrl} followPointer={followPointer} />
    </CatModelBoundary>
  )
}

export function CatScene({
  className,
  autoRotate = false,
  followPointer = true,
  modelUrl = LEAD_CAT_MODEL_URL,
}: CatSceneProps) {
  return (
    <div className={className} style={{ width: "100%", height: "100%" }}>
      <Canvas
        shadows
        dpr={[1, 2]}
        camera={{ position: [0, 1.05, 4.6], fov: 42 }}
        gl={{ antialias: true, alpha: true, preserveDrawingBuffer: true }}
      >
        <StagePlatform />
        <Float speed={1.8} rotationIntensity={0.08} floatIntensity={0.14}>
          <Suspense fallback={null}>
            <CatModel followPointer={followPointer} modelUrl={modelUrl} />
          </Suspense>
        </Float>
        <OrbitControls
          enableZoom={false}
          enablePan={false}
          autoRotate={autoRotate}
          autoRotateSpeed={0.6}
          target={[0, 0.9, 0]}
          minPolarAngle={Math.PI / 2.8}
          maxPolarAngle={Math.PI / 2.05}
        />
      </Canvas>
    </div>
  )
}
