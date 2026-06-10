import { Component, type ReactNode } from "react"

interface CatModelBoundaryProps {
  fallback: ReactNode
  children: ReactNode
}

interface CatModelBoundaryState {
  hasError: boolean
}

export class CatModelBoundary extends Component<
  CatModelBoundaryProps,
  CatModelBoundaryState
> {
  state: CatModelBoundaryState = { hasError: false }

  static getDerivedStateFromError(): CatModelBoundaryState {
    return { hasError: true }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback
    }
    return this.props.children
  }
}
