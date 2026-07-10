import { describe, expect, it } from 'vitest'
import {
  createWallEditorState,
  moveEndpoint,
  pushWallSnapshot,
  redo,
  undo,
} from './floorplanEditor'

describe('wall edit core', () => {
  it('moves only one endpoint when detached', () => {
    const state = createWallEditorState([
      {
        x1: 0,
        y1: 0,
        x2: 10,
        y2: 0,
      },
      {
        x1: 20,
        y1: 10,
        x2: 30,
        y2: 20,
      },
    ])

    const moved = moveEndpoint(state, { wallIndex: 0, endpoint: 'end' }, { x: 11, y: 1 })
    expect(moved.changed).toBe(true)
    expect(moved.walls[0].x2).toBe(11)
    expect(moved.walls[0].y2).toBe(1)
    expect(moved.walls[1].x1).toBe(20)
    expect(moved.walls[1].y1).toBe(10)
  })

  it('moves shared endpoints across adjacent walls', () => {
    const state = createWallEditorState([
      {
        x1: 0,
        y1: 0,
        x2: 10,
        y2: 0,
      },
      {
        x1: 10,
        y1: 0,
        x2: 20,
        y2: 0,
      },
      {
        x1: 30,
        y1: 5,
        x2: 40,
        y2: 5,
      },
    ], 6)

    const moved = moveEndpoint(state, { wallIndex: 0, endpoint: 'end' }, { x: 12, y: -1 })
    expect(moved.changed).toBe(true)
    expect(moved.walls[0].x2).toBe(12)
    expect(moved.walls[0].y2).toBe(-1)
    expect(moved.walls[1].x1).toBe(12)
    expect(moved.walls[1].y1).toBe(-1)
    expect(moved.walls[2].x1).toBe(30)
  })

  it('preserves proven shared topology after a committed move', () => {
    const initial = createWallEditorState([
      { x1: 0, y1: 0, x2: 10, y2: 0 },
      { x1: 10, y1: 0, x2: 20, y2: 0 },
    ])
    const firstMove = moveEndpoint(initial, { wallIndex: 0, endpoint: 'end' }, { x: 12, y: -1 })
    const committed = pushWallSnapshot(initial, firstMove.walls)

    const secondMove = moveEndpoint(committed, { wallIndex: 0, endpoint: 'end' }, { x: 14, y: -2 })

    expect(secondMove.walls[0]).toEqual({ x1: 0, y1: 0, x2: 14, y2: -2 })
    expect(secondMove.walls[1]).toEqual({ x1: 14, y1: -2, x2: 20, y2: 0 })
  })

  it('does not treat merely nearby endpoints as shared topology', () => {
    const state = createWallEditorState([
      { x1: 0, y1: 0, x2: 10, y2: 0 },
      { x1: 10.5, y1: 0, x2: 20, y2: 0 },
    ], 6)

    const moved = moveEndpoint(state, { wallIndex: 0, endpoint: 'end' }, { x: 12, y: -1 })

    expect(moved.changed).toBe(true)
    expect(moved.walls[0]).toEqual({ x1: 0, y1: 0, x2: 12, y2: -1 })
    expect(moved.walls[1]).toEqual({ x1: 10.5, y1: 0, x2: 20, y2: 0 })
  })

  it('ignores stale endpoint references without throwing', () => {
    const state = createWallEditorState([{ x1: 0, y1: 0, x2: 10, y2: 0 }])

    expect(moveEndpoint(state, { wallIndex: -1, endpoint: 'start' }, { x: 1, y: 1 })).toMatchObject({
      changed: false,
      walls: state.walls,
    })
    expect(moveEndpoint(state, { wallIndex: 1, endpoint: 'end' }, { x: 1, y: 1 })).toMatchObject({
      changed: false,
      walls: state.walls,
    })
  })

  it('supports undo and redo, and redo branch is truncated on new commit', () => {
    const initial = createWallEditorState([
      {
        x1: 0,
        y1: 0,
        x2: 10,
        y2: 0,
      },
    ])

    const step1 = moveEndpoint(initial, { wallIndex: 0, endpoint: 'end' }, { x: 10, y: 1 })
    const afterFirst = pushWallSnapshot(initial, step1.walls)
    const step2 = moveEndpoint(afterFirst, { wallIndex: 0, endpoint: 'start' }, { x: -2, y: 0 })
    const afterSecond = pushWallSnapshot(afterFirst, step2.walls)

    const undone = undo(afterSecond)
    expect(undone.walls[0].x2).toBe(10)
    expect(undone.walls[0].y2).toBe(1)
    expect(undone.undoStack.length).toBe(1)
    expect(undone.redoStack.length).toBe(1)

    const redone = redo(undone)
    expect(redone.walls[0].x1).toBe(-2)
    expect(redone.redoStack.length).toBe(0)

    const branched = pushWallSnapshot(redone, [
      {
        x1: -1,
        y1: 0,
        x2: 10,
        y2: 1,
      },
    ])
    expect(branched.redoStack.length).toBe(0)
    expect(branched.undoStack.length).toBe(3)
    expect(branched.walls[0].x1).toBe(-1)
  })

  it('ignores non-finite drag coordinates and keeps finite wall coordinates', () => {
    const state = createWallEditorState([
      {
        x1: 0,
        y1: 0,
        x2: 10,
        y2: 0,
      },
    ])

    const moved = moveEndpoint(state, { wallIndex: 0, endpoint: 'start' }, { x: Number.NaN, y: Infinity })
    expect(moved.changed).toBe(false)
    expect(moved.walls[0]).toEqual({ x1: 0, y1: 0, x2: 10, y2: 0 })
  })
})
