import { ContactShadows } from "@react-three/drei"
import { DoubleSide } from "three"

export function StagePlatform() {
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

      <ContactShadows
        position={[0, 0.01, 0]}
        opacity={0.38}
        scale={3.6}
        blur={2.5}
        far={2.5}
        resolution={1024}
        color="#7A4A2B"
      />
    </>
  )
}
