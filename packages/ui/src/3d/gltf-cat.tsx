import { useRef } from "react"
import { useFrame } from "@react-three/fiber"
import { useGLTF } from "@react-three/drei"
import type { Group } from "three"
import { MathUtils } from "three"

export interface GltfCatProps {
  src: string
  followPointer?: boolean
}

export function GltfCat({ src, followPointer = true }: GltfCatProps) {
  const root = useRef<Group>(null)
  const gltf = useGLTF(src)

  useFrame((state) => {
    const t = state.clock.elapsedTime
    if (!root.current) {
      return
    }
    root.current.position.y = Math.sin(t * 1.4) * 0.07
    const targetY = followPointer ? state.pointer.x * 0.45 : 0
    const targetX = followPointer ? state.pointer.y * 0.18 : 0
    root.current.rotation.y = MathUtils.lerp(
      root.current.rotation.y,
      targetY - 0.3,
      0.06
    )
    root.current.rotation.x = MathUtils.lerp(
      root.current.rotation.x,
      targetX,
      0.06
    )
  })

  return (
    <group ref={root} position={[0, -0.6, 0]}>
      <primitive object={gltf.scene} />
    </group>
  )
}
