import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-scan.test.mjs`。
/// 「取消扫码 / 拒绝相机权限」两条用 `uni` 相机接口，本轮只迁扫码文本之后的逻辑，未搬。
final class MatchScanTests: XCTestCase {
    func testLegacyProfileQRPayloadIsRejectedButSignedInviteShapeIsAccepted() {
        XCTAssertEqual(
            MatchScan.parsePayload(#"{"user_id":2002,"nickname":"可伪造昵称"}"#),
            .error(message: "该二维码已不再支持，请让对方刷新匹配二维码")
        )
        XCTAssertEqual(
            MatchScan.parsePayload("header.payload"),
            .previewMatchInvite(inviteToken: "header.payload")
        )
        guard case .error = MatchScan.parsePayload("headerXpayload") else {
            return XCTFail("多段 token 不应被当成匹配码")
        }
        guard case .error = MatchScan.parsePayload("not a code") else {
            return XCTFail("普通文本不应被当成匹配码")
        }
    }

    func testPreviewReturnsTrustedOpponentBeforeBuildingStartAction() async {
        let result = await MatchScan.resolvePayload(
            rawValue: "signed.invite",
            previewMatchInvite: { token in
                MatchInvitePreviewResponse(
                    success: true,
                    message: nil,
                    preview: MatchInvitePreviewInfo(
                        opponentId: 2002,
                        opponentName: "可信对手",
                        opponentAvatar: "trusted.png"
                    )
                )
            }
        )
        XCTAssertEqual(
            result,
            .startMatch(
                inviteToken: "signed.invite",
                opponent: MatchScanOpponent(id: 2002, nickname: "可信对手", avatar: "trusted.png")
            )
        )
    }

    func testRefereeJoinPreservesCredentialsAndExpiredInvitesExplainWhy() async {
        XCTAssertEqual(
            MatchScan.parsePayload(#"{"type":"match_referee","match_id":88,"join_token":"server-issued-join-token"}"#),
            .joinReferee(refereeJoin: RefereeJoin(matchId: 88, joinToken: "server-issued-join-token"))
        )
        // 凭据不进日志与打印。
        XCTAssertFalse(
            String(describing: MatchScan.parsePayload(
                #"{"type":"match_referee","match_id":88,"join_token":"server-issued-join-token"}"#
            )).contains("server-issued-join-token")
        )

        let expired = await MatchScan.resolvePayload(
            rawValue: "signed.expired",
            previewMatchInvite: { _ in
                MatchInvitePreviewResponse(success: false, message: "匹配二维码已失效，请让对方刷新二维码", preview: nil)
            }
        )
        XCTAssertEqual(expired, .error(message: "匹配二维码已失效，请让对方刷新二维码"))
    }

    func testEntryLinksAreSeparatedBeforeRefereeJSONAndBareTokens() {
        let code = "Ab3dEf6hIj9k"
        XCTAssertEqual(
            MatchScan.parsePayload("https://www.zhuifen.cn/go/?j=\(code)"),
            .resolveInviteCode(inviteCode: code)
        )
        XCTAssertEqual(
            MatchScan.parsePayload("https://www.zhuifen.cn/go/?m=123"),
            .openMatch(matchId: 123)
        )
        XCTAssertEqual(
            MatchScan.parsePayload("https://www.zhuifen.cn/go/"),
            .error(message: "无效的二维码")
        )
        XCTAssertEqual(
            MatchScan.parsePayload("https://example.com/go/?j=Ab3dEf6hIj9k"),
            .error(message: "无效的二维码")
        )
    }

    func testInviteLinkCodeResolvesIntoTheSameStartActionAsABareToken() async {
        let result = await MatchScan.resolvePayload(
            rawValue: "https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9k",
            resolveMatchInviteCode: { code in
                XCTAssertEqual(code, "Ab3dEf6hIj9k")
                return MatchInviteResolveResponse(
                    success: true,
                    message: nil,
                    inviteToken: "header.payload",
                    preview: MatchInvitePreviewInfo(
                        opponentId: 9,
                        opponentName: "对手乙",
                        opponentAvatar: "a.png"
                    )
                )
            }
        )
        XCTAssertEqual(
            result,
            .startMatch(
                inviteToken: "header.payload",
                opponent: MatchScanOpponent(id: 9, nickname: "对手乙", avatar: "a.png")
            )
        )
    }

    func testFailedInviteLinkExchangeReportsTheServerMessage() async {
        let expired = await MatchScan.resolvePayload(
            rawValue: "https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9k",
            resolveMatchInviteCode: { _ in
                MatchInviteResolveResponse(
                    success: false,
                    message: "匹配二维码已失效，请让对方刷新二维码",
                    inviteToken: nil,
                    preview: nil
                )
            }
        )
        XCTAssertEqual(expired, .error(message: "匹配二维码已失效，请让对方刷新二维码"))

        let broken = await MatchScan.resolvePayload(
            rawValue: "https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9k",
            resolveMatchInviteCode: { _ in throw URLError(.notConnectedToInternet) }
        )
        XCTAssertEqual(broken, .error(message: "匹配二维码服务暂不可用，请稍后重试"))

        let missing = await MatchScan.resolvePayload(rawValue: "https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9k")
        XCTAssertEqual(missing, .error(message: "匹配二维码服务暂不可用"))
    }
}
