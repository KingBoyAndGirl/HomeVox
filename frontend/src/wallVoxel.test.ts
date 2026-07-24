import { describe, expect, it } from 'vitest'
import { buildWallVoxelModel, WALL_VOXEL_GRID_SIZE } from './wallVoxel'

function voxelAt(model: NonNullable<ReturnType<typeof buildWallVoxelModel>>, xIndex: number, yIndex: number, zIndex: number): number {
  const [nx, ny] = model.dimensions
  return model.data[xIndex + yIndex * nx + zIndex * nx * ny]
}

describe('buildWallVoxelModel', () => {
  it('creates a finite, controlled 17³ field from editable walls', () => {
    const model = buildWallVoxelModel([{ x1: 0, y1: 0, x2: 300, y2: 0 }])

    expect(model).not.toBeNull()
    expect(model?.dimensions).toEqual([WALL_VOXEL_GRID_SIZE, WALL_VOXEL_GRID_SIZE, WALL_VOXEL_GRID_SIZE])
    expect(model?.data).toHaveLength(WALL_VOXEL_GRID_SIZE ** 3)
    expect(Array.from(model?.data ?? []).every(Number.isFinite)).toBe(true)
    expect(model?.spacing.every((value) => value > 0)).toBe(true)
  })

  it('rejects empty, non-finite, and degenerate walls', () => {
    expect(buildWallVoxelModel([])).toBeNull()
    expect(buildWallVoxelModel([{ x1: 0, y1: 0, x2: 0, y2: 0 }])).toBeNull()
    expect(buildWallVoxelModel([{ x1: 0, y1: 0, x2: Number.NaN, y2: 1 }])).toBeNull()
  })

  it('subtracts a vertical wall door along the wall tangent rather than world X', () => {
    const model = buildWallVoxelModel(
      [{ id: 'vertical', x1: 0, y1: 0, x2: 0, y2: 300 }],
      [{ id: 'door-vertical', wallId: 'vertical', position: 0.7, width: 60, kind: 'door' }],
    )

    expect(model).not.toBeNull()
    // x=0, z≈2.68 is inside the wall-local door span but outside a world-X cut.
    expect(voxelAt(model!, 8, 7, 12)).toBeLessThan(0)
  })

  it('subtracts a diagonal wall door along the wall tangent rather than world axes', () => {
    const model = buildWallVoxelModel(
      [{ id: 'diagonal', x1: 0, y1: 0, x2: 300, y2: 300 }],
      [{ id: 'door-diagonal', wallId: 'diagonal', position: 0.7, width: 80, kind: 'door' }],
    )

    expect(model).not.toBeNull()
    // x=z≈2.01 is inside the diagonal wall-local door span but outside a world-axis cut.
    expect(voxelAt(model!, 11, 7, 11)).toBeLessThan(0)
  })

  it('cuts a finite window opening from the wall field rather than only adding a marker', () => {
    const walls = [{ id: 'wall-a', x1: 0, y1: 0, x2: 300, y2: 0 }]
    const solid = buildWallVoxelModel(walls)
    const withWindow = buildWallVoxelModel(
      walls,
      [],
      [{ id: 'window-a', kind: 'window', wallId: 'wall-a', position: 0.5, width: 70, confirmed: false }],
    )

    expect(solid).not.toBeNull()
    expect(withWindow).not.toBeNull()
    expect(Array.from(withWindow!.data).every(Number.isFinite)).toBe(true)
    expect(Array.from(withWindow!.data)).not.toEqual(Array.from(solid!.data))
  })

  it('keeps multiple legal near-end openings finite while changing the generated field', () => {
    const walls = [{ id: 'wall-a', x1: 0, y1: 0, x2: 300, y2: 0 }]
    const oneOpening = buildWallVoxelModel(
      walls,
      [{ id: 'door-a', kind: 'door', wallId: 'wall-a', position: 0.15, width: 60, confirmed: false }],
    )
    const multipleOpenings = buildWallVoxelModel(
      walls,
      [
        { id: 'door-a', kind: 'door', wallId: 'wall-a', position: 0.15, width: 60, confirmed: false },
        { id: 'door-b', kind: 'door', wallId: 'wall-a', position: 0.85, width: 60, confirmed: false },
      ],
      [{ id: 'window-a', kind: 'window', wallId: 'wall-a', position: 0.5, width: 60, confirmed: false }],
    )

    expect(oneOpening).not.toBeNull()
    expect(multipleOpenings).not.toBeNull()
    expect(Array.from(multipleOpenings!.data).every(Number.isFinite)).toBe(true)
    expect(Array.from(multipleOpenings!.data)).not.toEqual(Array.from(oneOpening!.data))
  })

  it('ignores non-finite opening data without producing a non-finite field', () => {
    const model = buildWallVoxelModel(
      [{ id: 'wall-a', x1: 0, y1: 0, x2: 300, y2: 0 }],
      [{ id: 'invalid', kind: 'door', wallId: 'wall-a', position: Number.NaN, width: 60 }],
    )

    expect(model).not.toBeNull()
    expect(Array.from(model!.data).every(Number.isFinite)).toBe(true)
  })
})
