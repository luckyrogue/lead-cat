import { Suspense } from "react"
import { Canvas } from "@react-three/fiber"
import {
  ContactShadows,
  Environment,
  Float,
  OrbitControls,
} from "@react-three/drei"

import { CatModelBoundary } from "./cat-model-boundary"
import { GltfCat } from "./gltf-cat"
import { LowPolyCat } from "./low-poly-cat"

export interface CatSceneProps {
  className?: string
  autoRotate?: boolean
  followPointer?: boolean
  modelUrl?: string
}

export function CatScene({
  className,
  autoRotate = false,
  followPointer = true,
  modelUrl = "/models/lead-cat.glb",
}: CatSceneProps) {
  return (
    <div className={className} style={{ width: "100%", height: "100%" }}>
      <Canvas
        shadows
        dpr={[1, 2]}
        camera={{ position: [0, 0.6, 5], fov: 42 }}
        gl={{ antialias: true, alpha: true }}
      >
        <ambientLight intensity={0.85} color="#FFEFE0" />
        <directionalLight
          position={[3, 5, 4]}
          intensity={1.6}
          color="#FFD9A0"
          castShadow
          shadow-mapSize={[1024, 1024]}
        />
        <directionalLight
          position={[-4, 2, -2]}
          intensity={0.5}
          color="#F2603F"
        />

        <Float speed={1.4} rotationIntensity={0.25} floatIntensity={0.5}>
          <CatModelBoundary
            fallback={<LowPolyCat followPointer={followPointer} />}
          >
            <Suspense
              fallback={<LowPolyCat followPointer={followPointer} />}
            >
              <GltfCat src={modelUrl} followPointer={followPointer} />
            </Suspense>
          </CatModelBoundary>
        </Float>

        <ContactShadows
          position={[0, -1.3, 0]}
          opacity={0.35}
          scale={6}
          blur={2.6}
          far={3}
          color="#C68B59"
        />
        <Environment preset="sunset" />
        <OrbitControls
          enableZoom={false}
          enablePan={false}
          autoRotate={autoRotate}
          autoRotateSpeed={1.2}
          minPolarAngle={Math.PI / 3}
          maxPolarAngle={Math.PI / 1.8}
        />
      </Canvas>
    </div>
  )
}
