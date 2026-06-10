import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import type { Group, Mesh } from "three";
import { MathUtils } from "three";

const FUR = "#C68B59";
const FUR_LIGHT = "#E6C39C";
const BELLY = "#FAF1E8";
const INNER_EAR = "#FFB199";
const NOSE = "#FF6F61";
const EYE = "#3A2A1C";

export interface LowPolyCatProps {
  followPointer?: boolean;
}

export function LowPolyCat({ followPointer = true }: LowPolyCatProps) {
  const root = useRef<Group>(null);
  const head = useRef<Group>(null);
  const tail = useRef<Group>(null);
  const leftEar = useRef<Mesh>(null);
  const rightEar = useRef<Mesh>(null);

  useFrame((state) => {
    const t = state.clock.elapsedTime;

    if (root.current) {
      root.current.position.y = Math.sin(t * 1.4) * 0.07;
      root.current.scale.setScalar(1 + Math.sin(t * 1.4) * 0.012);
    }

    if (head.current) {
      const targetX = followPointer ? state.pointer.y * 0.18 : 0;
      const targetY = followPointer ? state.pointer.x * 0.45 : 0;
      head.current.rotation.x = MathUtils.lerp(head.current.rotation.x, targetX, 0.08);
      head.current.rotation.y = MathUtils.lerp(head.current.rotation.y, targetY, 0.08);
      head.current.position.y = 0.95 + Math.sin(t * 1.4) * 0.02;
    }

    if (tail.current) {
      tail.current.rotation.z = Math.sin(t * 1.6) * 0.4 - 0.3;
      tail.current.rotation.x = Math.sin(t * 1.1) * 0.18;
    }

    const earTwitch = Math.sin(t * 2.3) * 0.06;
    if (leftEar.current) leftEar.current.rotation.z = 0.35 + earTwitch;
    if (rightEar.current) rightEar.current.rotation.z = -0.35 - earTwitch;
  });

  return (
    <group ref={root} position={[0, -0.4, 0]} rotation={[0, -0.3, 0]}>
      <mesh position={[0, 0.1, 0]} castShadow>
        <sphereGeometry args={[0.85, 6, 5]} />
        <meshStandardMaterial color={FUR} flatShading />
      </mesh>
      <mesh position={[0, -0.15, 0.55]} scale={[0.6, 0.7, 0.4]}>
        <sphereGeometry args={[0.7, 6, 5]} />
        <meshStandardMaterial color={BELLY} flatShading />
      </mesh>

      <group ref={tail} position={[0, 0.1, -0.75]}>
        <mesh position={[0, 0.45, -0.1]} rotation={[0.3, 0, 0]} castShadow>
          <capsuleGeometry args={[0.13, 1.1, 3, 6]} />
          <meshStandardMaterial color={FUR_LIGHT} flatShading />
        </mesh>
      </group>

      {[
        [-0.4, -0.78, 0.45],
        [0.4, -0.78, 0.45],
        [-0.4, -0.78, -0.35],
        [0.4, -0.78, -0.35],
      ].map((p, i) => (
        <mesh key={i} position={p as [number, number, number]} castShadow>
          <sphereGeometry args={[0.2, 5, 4]} />
          <meshStandardMaterial color={FUR_LIGHT} flatShading />
        </mesh>
      ))}

      <group ref={head} position={[0, 0.95, 0.25]}>
        <mesh castShadow>
          <sphereGeometry args={[0.7, 6, 5]} />
          <meshStandardMaterial color={FUR} flatShading />
        </mesh>
        <mesh position={[0, -0.1, 0.5]} scale={[0.85, 0.7, 0.6]}>
          <sphereGeometry args={[0.45, 6, 5]} />
          <meshStandardMaterial color={BELLY} flatShading />
        </mesh>

        <mesh ref={leftEar} position={[-0.4, 0.6, 0]} rotation={[0, 0, 0.35]} castShadow>
          <coneGeometry args={[0.28, 0.5, 4]} />
          <meshStandardMaterial color={FUR} flatShading />
        </mesh>
        <mesh ref={rightEar} position={[0.4, 0.6, 0]} rotation={[0, 0, -0.35]} castShadow>
          <coneGeometry args={[0.28, 0.5, 4]} />
          <meshStandardMaterial color={FUR} flatShading />
        </mesh>
        <mesh position={[-0.4, 0.55, 0.05]} rotation={[0, 0, 0.35]}>
          <coneGeometry args={[0.15, 0.3, 4]} />
          <meshStandardMaterial color={INNER_EAR} flatShading />
        </mesh>
        <mesh position={[0.4, 0.55, 0.05]} rotation={[0, 0, -0.35]}>
          <coneGeometry args={[0.15, 0.3, 4]} />
          <meshStandardMaterial color={INNER_EAR} flatShading />
        </mesh>

        <mesh position={[-0.25, 0.08, 0.62]}>
          <sphereGeometry args={[0.13, 6, 5]} />
          <meshStandardMaterial color={EYE} flatShading />
        </mesh>
        <mesh position={[0.25, 0.08, 0.62]}>
          <sphereGeometry args={[0.13, 6, 5]} />
          <meshStandardMaterial color={EYE} flatShading />
        </mesh>
        <mesh position={[-0.21, 0.13, 0.73]}>
          <sphereGeometry args={[0.045, 5, 4]} />
          <meshStandardMaterial color="#ffffff" />
        </mesh>
        <mesh position={[0.29, 0.13, 0.73]}>
          <sphereGeometry args={[0.045, 5, 4]} />
          <meshStandardMaterial color="#ffffff" />
        </mesh>

        <mesh position={[0, -0.12, 0.72]} rotation={[Math.PI, 0, 0]}>
          <coneGeometry args={[0.08, 0.1, 4]} />
          <meshStandardMaterial color={NOSE} flatShading />
        </mesh>
      </group>
    </group>
  );
}
