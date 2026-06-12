import { useMemo, useRef } from "react"
import { useFrame } from "@react-three/fiber"
import { useGLTF } from "@react-three/drei"
import type { Group, Material, Mesh } from "three"
import { Box3, MathUtils, Vector3 } from "three"

export interface GltfCatProps {
  src: string
  followPointer?: boolean
}

const TARGET_HEIGHT = 2.2

function prepareScene(scene: Group) {
  const model = scene.clone(true)

  const box = new Box3().setFromObject(model)
  const size = new Vector3()
  const center = new Vector3()
  box.getSize(size)
  box.getCenter(center)

  const scale = size.y > 0 ? TARGET_HEIGHT / size.y : 1
  model.scale.setScalar(scale)

  model.position.x = -center.x * scale
  model.position.z = -center.z * scale
  model.position.y = -box.min.y * scale

  model.traverse((child) => {
    const mesh = child as Mesh
    if (!mesh.isMesh) {
      return
    }

    mesh.castShadow = true
    mesh.receiveShadow = true

    const sourceMaterials = Array.isArray(mesh.material)
      ? mesh.material
      : [mesh.material]

    const materials = sourceMaterials.map((material) => {
      const next = material.clone() as Material & { flatShading?: boolean }
      if ("flatShading" in next) {
        next.flatShading = true
      }
      return next
    })

    mesh.material = materials.length === 1 ? materials[0] : materials
  })

  return model
}

export function GltfCat({ src, followPointer = true }: GltfCatProps) {
  const root = useRef<Group>(null)
  const { scene } = useGLTF(src)
  const model = useMemo(() => prepareScene(scene), [scene])

  useFrame((state) => {
    const t = state.clock.elapsedTime
    if (!root.current) {
      return
    }

    root.current.position.y = Math.sin(t * 1.4) * 0.03
    const targetY = followPointer ? state.pointer.x * 0.4 : 0
    root.current.rotation.y = MathUtils.lerp(
      root.current.rotation.y,
      targetY - 0.25,
      0.05,
    )
  })

  return (
    <group ref={root} position={[0, 0, 0]}>
      <primitive object={model} />
    </group>
  )
}
