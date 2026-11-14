import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/gear-match.test.mjs` 的 13 条断言。
@MainActor
final class GearMatchRulesTests: XCTestCase {
    func testOnlyCompletedMatchesViewedByAPlayerRequestTheGearRecord() {
        XCTAssertTrue(GearMatchRules.shouldShowMatchGear(status: 2, viewerRole: "player1"))
        XCTAssertTrue(GearMatchRules.shouldShowMatchGear(status: 2, viewerRole: "player2"))
        XCTAssertTrue(GearMatchRules.shouldShowMatchGear(status: 2, isPlayer1: true))
        XCTAssertFalse(GearMatchRules.shouldShowMatchGear(status: 2, viewerRole: "referee"))
        // 参与者详情可能把裁判的 is_player1 标成 true，裁判仍需保持在用杆接口之外。
        XCTAssertFalse(GearMatchRules.shouldShowMatchGear(status: 2, viewerRole: "referee", isPlayer1: true))
        XCTAssertFalse(GearMatchRules.shouldShowMatchGear(status: 2, viewerRole: "spectator"))
        XCTAssertFalse(GearMatchRules.shouldShowMatchGear(status: 2))
        XCTAssertFalse(GearMatchRules.shouldShowMatchGear(status: 1, viewerRole: "player1"))
        XCTAssertFalse(GearMatchRules.shouldShowMatchGear(status: 2, viewerRole: "player1", isSpectate: true))
        XCTAssertFalse(GearMatchRules.shouldShowMatchGear(status: nil))
    }

    func testRefusalsThatMeanNothingToShowHideTheCard() {
        for reason in ["not_participant", "not_completed", "not_found"] {
            XCTAssertTrue(GearMatchRules.isHiddenResponse(response(success: false, reason: reason)))
        }
        XCTAssertFalse(GearMatchRules.isHiddenResponse(response(success: false, reason: "")))
        XCTAssertFalse(GearMatchRules.isHiddenResponse(response(success: false, message: "读取本场用杆失败")))
        XCTAssertFalse(GearMatchRules.isHiddenResponse(response(success: true)))
        XCTAssertFalse(GearMatchRules.isHiddenResponse(nil))
    }

    func testUnrecordedSideHasNoRowsAndRecordedSideListsFourSlotsInOrder() {
        XCTAssertEqual(GearMatchRules.describeSide(nil), GearMatchSideView(recorded: false, rows: []))
        let empty = GearMatchRules.describeSide(side())
        XCTAssertFalse(empty.recorded)
        XCTAssertTrue(empty.rows.isEmpty)

        let data = side(
            recorded: true,
            cue: item(id: 1, category: "cue", brand: "Alpha", model: "A-100", cueStyle: "pool"),
            tipProduct: product(id: 5, brand: "Alpha", name: "A-Tip"),
            chalkProduct: product(id: 6, brand: "", name: "球房巧克"),
            breakCue: item(
                id: 2, category: "break_jump", brand: "Alpha", model: "A-Break", roles: ["break"]
            )
        )
        let view = GearMatchRules.describeSide(data)
        XCTAssertTrue(view.recorded)
        XCTAssertEqual(view.rows.map { [$0.key, $0.label, $0.title] }, [
            ["cue", "球杆", "Alpha A-100"],
            ["tip", "皮头", "Alpha A-Tip"],
            ["chalk", "巧克", "球房巧克"],
            ["break", "冲杆", "Alpha A-Break"],
        ])
        XCTAssertEqual(view.rows[0].subtitle, "美式杆")

        // 记录里只有球杆：不凭空展示其他行。
        let cueOnly = GearMatchRules.describeSide(side(recorded: true, cue: data.cue))
        XCTAssertEqual(cueOnly.rows.map(\.key), ["cue"])
    }

    func testEditDeadlineReadsAsMonthAndDay() {
        XCTAssertEqual(GearMatchRules.formatEditDeadline("2026-05-03T12:00:00+08:00"), "5月3日前可修改")
        XCTAssertEqual(GearMatchRules.formatEditDeadline("2026-12-31T08:00:00+08:00"), "12月31日前可修改")
        XCTAssertEqual(GearMatchRules.formatEditDeadline(""), "")
        XCTAssertEqual(GearMatchRules.formatEditDeadline("not a date"), "")
        XCTAssertEqual(GearMatchRules.formatEditDeadline(nil), "")
    }

    func testEditingNeedsTheServerFlagAndAnUnexpiredDeadline() throws {
        let deadlineText = "2026-05-03T00:00:00+08:00"
        let deadline = try XCTUnwrap(AppFormat.parseDateTime(deadlineText))
        let data = response(editable: true, deadline: deadlineText)

        XCTAssertTrue(GearMatchRules.canEdit(data, now: deadline.addingTimeInterval(-1)))
        XCTAssertFalse(GearMatchRules.canEdit(data, now: deadline), "到点即不可改")
        XCTAssertFalse(GearMatchRules.canEdit(data, now: deadline.addingTimeInterval(1)))

        var notEditable = data
        notEditable.editable = false
        XCTAssertFalse(GearMatchRules.canEdit(notEditable, now: deadline.addingTimeInterval(-1)))
        XCTAssertFalse(GearMatchRules.canEdit(response(success: false), now: Date(timeIntervalSince1970: 0)))
        XCTAssertFalse(GearMatchRules.canEdit(nil))
        // 没有可解析的截止时间时信任服务端的判断。
        XCTAssertTrue(GearMatchRules.canEdit(
            response(editable: true, deadline: ""),
            now: Date(timeIntervalSince1970: 123)
        ))
    }

    func testDraftStartsFromCurrentRecordAndSwitchingCuesPrefillsItsParts() {
        let data = side(
            recorded: true,
            cue: item(id: 1, category: "cue"),
            tipProduct: product(id: 5),
            chalkProduct: product(id: 6),
            breakCue: item(id: 2, category: "break_jump")
        )
        let draft = GearMatchRules.draft(from: data)
        XCTAssertEqual(draft.cueItemId, 1)
        XCTAssertEqual(draft.tipProduct?.id, 5)
        XCTAssertEqual(draft.chalkProduct?.id, 6)
        XCTAssertEqual(draft.breakItemId, 2)

        let empty = GearMatchRules.draft(from: side())
        XCTAssertEqual(empty, GearMatchDraft(cueItemId: 0, tipProduct: nil, chalkProduct: nil, breakItemId: 0))

        var other = item(id: 9, category: "cue")
        other.tipProduct = product(id: 15)
        other.chalkProduct = nil
        let switched = GearMatchRules.applyCue(other, to: draft)
        XCTAssertEqual(switched.cueItemId, 9)
        XCTAssertEqual(switched.tipProduct?.id, 15)
        XCTAssertNil(switched.chalkProduct, "新球杆没有巧克就清空，而不是沿用旧的")
        XCTAssertEqual(switched.breakItemId, 2, "换球杆不影响冲杆")

        let cleared = GearMatchRules.applyCue(nil, to: draft)
        XCTAssertEqual(cleared.cueItemId, 0)
        XCTAssertNil(cleared.tipProduct)
    }

    func testRequestBodyCarriesIdsOnlyAndDropsTheBreakCueWhereThereIsNone() {
        let draft = GearMatchDraft(
            cueItemId: 1,
            tipProduct: product(id: 5),
            chalkProduct: nil,
            breakItemId: 2
        )
        let payload = GearMatchRules.payload(matchId: 77, draft: draft)
        XCTAssertEqual(payload.matchId, 77)
        XCTAssertEqual(payload.cueItemId, 1)
        XCTAssertEqual(payload.tipProductId, 5)
        XCTAssertEqual(payload.chalkProductId, 0)
        XCTAssertEqual(payload.breakItemId, 2)

        XCTAssertEqual(GearMatchRules.payload(matchId: 77, draft: draft, supportsBreak: false).breakItemId, 0)
        XCTAssertTrue(GearMatchRules.isDraftEmpty(GearMatchRules.draft(from: nil)))
        XCTAssertFalse(GearMatchRules.isDraftEmpty(draft))
        XCTAssertFalse(GearMatchRules.isDraftEmpty(GearMatchDraft(
            cueItemId: 0,
            tipProduct: nil,
            chalkProduct: product(id: 6),
            breakItemId: 0
        )))
    }

    func testEditorOptionsIncludeRetiredCuesButNeverDeletedOnesAndBreakNeedsTheRole() {
        let items = [
            item(id: 1, category: "cue", status: 1),
            item(id: 2, category: "cue", status: 2),
            item(id: 3, category: "cue", status: 3),
            item(id: 4, category: "break_jump", status: 1, roles: ["break"]),
            item(id: 5, category: "break_jump", status: 2, roles: ["break", "jump"]),
            item(id: 6, category: "break_jump", status: 1, roles: ["jump"]),
            item(id: 7, category: "break_jump", status: 3, roles: ["break"]),
        ]

        XCTAssertEqual(GearMatchRules.cueOptions(items).map(\.id), [1, 2])
        XCTAssertEqual(GearMatchRules.breakOptions(items).map(\.id), [4, 5])
        XCTAssertEqual(GearMatchRules.cueOptions([]).count, 0)
    }

    // MARK: - 读取器：首次进入、重复读取、切号与迟到响应

    func testFirstReadReturnsItsResponseAndCarriesTheMatchId() async throws {
        let loader = GearMatchLoader(
            request: { matchId in
                var response = GearMatchResponse()
                response.success = true
                response.matchId = matchId
                return response
            },
            readIdentity: { GearMatchLoader.Identity(userId: 1, generation: 1) }
        )

        let result = try await loader.load(500)

        XCTAssertFalse(result.stale)
        XCTAssertEqual(result.response.matchId, 500)
    }

    func testOnlyTheNewestOfTwoOverlappingReadsIsAccepted() async throws {
        let firstGate = Gate()
        let secondGate = Gate()
        let counter = Counter()
        let loader = GearMatchLoader(
            request: { _ in
                let call = counter.next()
                return call == 1 ? try await firstGate.wait() : try await secondGate.wait()
            },
            readIdentity: { GearMatchLoader.Identity(userId: 1, generation: 1) }
        )

        let first = Task { try await loader.load(500) }
        let second = Task { try await loader.load(500) }
        await waitUntil { counter.current == 2 }

        secondGate.resolve(response(matchId: 2))
        firstGate.resolve(response(matchId: 1))

        let firstResult = try await first.value
        let secondResult = try await second.value
        XCTAssertTrue(firstResult.stale, "较早的读取不能覆盖较新的结果")
        XCTAssertFalse(secondResult.stale)
        XCTAssertEqual(secondResult.response.matchId, 2)
    }

    func testResponseArrivingAfterAccountSwitchIsDropped() async throws {
        let gate = Gate()
        let started = Counter()
        let box = GearIdentityBox(userId: 1, generation: 1)
        let loader = GearMatchLoader(
            request: { _ in
                _ = started.next()
                return try await gate.wait()
            },
            readIdentity: { box.identity }
        )

        let pending = Task { try await loader.load(500) }
        await waitUntil { started.current == 1 }
        box.userId = 2
        box.generation = 2
        gate.resolve(response())

        let result = try await pending.value
        XCTAssertTrue(result.stale)
    }

    func testResponseAfterAuthGenerationChangeOrCancelIsDropped() async throws {
        let gate = Gate()
        let started = Counter()
        let box = GearIdentityBox(userId: 1, generation: 1)
        let loader = GearMatchLoader(
            request: { _ in
                _ = started.next()
                return try await gate.wait()
            },
            readIdentity: { box.identity }
        )

        let pending = Task { try await loader.load(500) }
        await waitUntil { started.current == 1 }
        box.generation = 9
        gate.resolve(response())
        let generationChanged = try await pending.value
        XCTAssertTrue(generationChanged.stale)

        let cancelledGate = Gate()
        let cancelledStarted = Counter()
        let second = GearMatchLoader(
            request: { _ in
                _ = cancelledStarted.next()
                return try await cancelledGate.wait()
            },
            readIdentity: { GearMatchLoader.Identity(userId: 1, generation: 1) }
        )
        let cancelled = Task { try await second.load(500) }
        await waitUntil { cancelledStarted.current == 1 }
        second.cancel()
        cancelledGate.resolve(response())
        let cancelledResult = try await cancelled.value
        XCTAssertTrue(cancelledResult.stale)
    }

    func testFailedReadThrowsSoTheCardCanShowARetryState() async {
        let loader = GearMatchLoader(
            request: { _ in throw GearTestError.network },
            readIdentity: { GearMatchLoader.Identity(userId: 1, generation: 1) }
        )

        do {
            _ = try await loader.load(500)
            XCTFail("读取失败应抛错")
        } catch {
            XCTAssertEqual(error.localizedDescription, "网络错误")
        }
    }

    // MARK: - 夹具

    private func item(
        id: Int64,
        category: String,
        status: Int = 1,
        brand: String = "",
        model: String = "",
        cueStyle: String? = nil,
        roles: [String]? = nil
    ) -> GearItemInfo {
        var item = GearItemInfo()
        item.id = id
        item.category = category
        item.status = status
        item.brandName = brand
        item.modelName = model
        item.cueStyle = cueStyle
        item.roles = roles
        return item
    }

    private func product(id: Int64, brand: String = "", name: String = "") -> GearProductInfo {
        var product = GearProductInfo()
        product.id = id
        product.brandName = brand
        product.name = name
        return product
    }

    private func side(
        recorded: Bool = false,
        cue: GearItemInfo? = nil,
        tipProduct: GearProductInfo? = nil,
        chalkProduct: GearProductInfo? = nil,
        breakCue: GearItemInfo? = nil
    ) -> GearMatchSide {
        var side = GearMatchSide()
        side.recorded = recorded
        side.cue = cue
        side.tipProduct = tipProduct
        side.chalkProduct = chalkProduct
        side.breakCue = breakCue
        return side
    }

    private func response(
        success: Bool = true,
        message: String? = nil,
        reason: String? = nil,
        matchId: Int64 = 0,
        editable: Bool = false,
        deadline: String? = nil
    ) -> GearMatchResponse {
        var response = GearMatchResponse()
        response.success = success
        response.message = message
        response.reason = reason
        response.matchId = matchId
        response.editable = editable
        response.editDeadline = deadline
        return response
    }
}

@MainActor
private final class Gate {
    private var continuations: [CheckedContinuation<GearMatchResponse, Error>] = []

    func wait() async throws -> GearMatchResponse {
        try await withCheckedThrowingContinuation { continuations.append($0) }
    }

    func resolve(_ response: GearMatchResponse) {
        continuations.removeFirst().resume(returning: response)
    }
}

private final class Counter: @unchecked Sendable {
    private let lock = NSLock()
    private var value = 0

    @discardableResult
    func next() -> Int {
        lock.lock()
        defer { lock.unlock() }
        value += 1
        return value
    }

    var current: Int {
        lock.lock()
        defer { lock.unlock() }
        return value
    }
}

@MainActor
private final class GearIdentityBox {
    var userId: Int64?
    var generation: Int?

    init(userId: Int64?, generation: Int?) {
        self.userId = userId
        self.generation = generation
    }

    var identity: GearMatchLoader.Identity {
        GearMatchLoader.Identity(userId: userId, generation: generation)
    }
}

private enum GearTestError: LocalizedError {
    case network

    var errorDescription: String? { "网络错误" }
}
