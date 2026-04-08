import test from 'node:test'
import assert from 'node:assert/strict'

import { resolvePlayingViewerUi } from '../utils/match-role-view.js'

test('resolvePlayingViewerUi keeps referee as the sole writer', () => {
  assert.deepEqual(
    resolvePlayingViewerUi({
      viewerRole: 'referee',
      canScore: true,
      canUndo: true,
      canFinish: true,
      refereeName: '助教丙'
    }),
    {
      leftIdentity: '选手1',
      rightIdentity: '选手2',
      subtitleSuffix: '左选手1右选手2',
      roleLabel: '裁判视角',
      showActionPanel: true,
      showUndoButton: true,
      showFinishButton: true,
      readonlyHint: ''
    }
  )
})

test('resolvePlayingViewerUi turns players into readonly viewers after referee binding', () => {
  assert.deepEqual(
    resolvePlayingViewerUi({
      viewerRole: 'player1',
      canScore: false,
      canUndo: false,
      canFinish: false,
      refereeName: '助教丙'
    }),
    {
      leftIdentity: '我方',
      rightIdentity: '对手',
      subtitleSuffix: '左我右敌',
      roleLabel: '选手视角',
      showActionPanel: false,
      showUndoButton: false,
      showFinishButton: false,
      readonlyHint: '本场由助教丙负责记分'
    }
  )
})
