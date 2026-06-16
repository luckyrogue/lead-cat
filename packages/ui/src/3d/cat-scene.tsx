import { Suspense } from "react"
import { Canvas } from "@react-three/fiber"
import { ContactShadows, Float, OrbitControls, useGLTF } from "@react-three/drei"
import { DoubleSide } from "three"

import { CatModelBoundary } from "./cat-model-boundary"
import { GltfCat } from "./gltf-cat"
import { LowPolyCat } from "./low-poly-cat"

const DEFAULT_MODEL_URL = "/models/lead-cat.glb"

if (typeof window !== "undefined") {
  useGLTF.preload(DEFAULT_MODEL_URL)
}

export interface CatSceneProps {
  className?: string
  autoRotate?: boolean
  followPointer?: boolean
  modelUrl?: string
}

function CatStageContent({
  autoRotate,
  followPointer,
  modelUrl,
}: Required<Pick<CatSceneProps, "autoRotate" | "followPointer" | "modelUrl">>) {
  return (
    <>
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

      <mesh position={[0, -0.02, 0]} rotation={[-Math.PI / 2, 0, 0]} receiveShadow>
        <circleGeometry args={[1.5, 48]} />
        <meshStandardMaterial
          color="#FBE3CF"
          roughness={0.95}
          metalness={0}
          side={DoubleSide}
        />
      </mesh>
      <mesh position={[0, -0.06, 0]}>
        <cylinderGeometry args={[1.5, 1.6, 0.12, 48]} />
        <meshStandardMaterial color="#F6D2B6" roughness={1} metalness={0} />
      </mesh>

      <Float speed={1.8} rotationIntensity={0.08} floatIntensity={0.14}>
        <CatModelBoundary
          fallback={<LowPolyCat followPointer={followPointer} />}
        >
          <Suspense fallback={<LowPolyCat followPointer={followPointer} />}>
            <GltfCat src={modelUrl} followPointer={followPointer} />
          </Suspense>
        </CatModelBoundary>
      </Float>

      <ContactShadows
        position={[0, 0.01, 0]}
        opacity={0.38}
        scale={3.6}
        blur={2.5}
        far={2.5}
        resolution={1024}
        color="#7A4A2B"
      />
      <OrbitControls
        enableZoom={false}
        enablePan={false}
        autoRotate={autoRotate}
        autoRotateSpeed={0.6}
        target={[0, 0.9, 0]}
        minPolarAngle={Math.PI / 2.8}
        maxPolarAngle={Math.PI / 2.05}
      />
    </>
  )
}

export function CatScene({
  className,
  autoRotate = false,
  followPointer = true,
  modelUrl = DEFAULT_MODEL_URL,
}: CatSceneProps) {
  return (
    <div className={className} style={{ width: "100%", height: "100%" }}>
      <Canvas
        shadows
        dpr={[1, 2]}
        camera={{ position: [0, 1.05, 4.6], fov: 42 }}
        gl={{ antialias: true, alpha: true, preserveDrawingBuffer: true }}
      >
        <Suspense fallback={<LowPolyCat followPointer={followPointer} />}>
          <CatStageContent
            autoRotate={autoRotate}
            followPointer={followPointer}
            modelUrl={modelUrl}
          />
        </Suspense>
      </Canvas>
    </div>
  )
}
