var qrv;
var upvote = false;

$(function(){
    qrv = new QRCode(document.getElementById("voteQrcode"), "");
    $("#navSearch").removeClass("active");
    $("#upvote").click(function( event ) {
        clearVoteModal();
        upvote = true;
        $("#voteUp").removeClass("thumb");
        $("#voteUp").addClass("upvote");
        $('#voteModal').modal();
    });
    $("#downvote").click(function( event ) {
        clearVoteModal();
        upvote = false;
        $("#voteDown").removeClass("thumb");
        $("#voteDown").addClass("downvote");
        $('#voteModal').modal();
    });
    $("#voteUp").click(function( event ) {
        upvote = true;
        $("#voteDown").removeClass("downvote");
        $("#voteDown").addClass("thumb");
        $("#voteUp").removeClass("thumb");
        $("#voteUp").addClass("upvote");
    });
    $("#voteDown").click(function( event ) {
        upvote = false;
        $("#voteDown").removeClass("thumb");
        $("#voteDown").addClass("downvote");
        $("#voteUp").removeClass("upvote");
        $("#voteUp").addClass("thumb");
    });

    $("#comment").on('change keyup paste', function() {
        var comment = $("#comment").val();
        var l = lengthInUtf8Bytes(comment);
        $("#commentRemainingChars").text(177 - l + " characters remaining");
        maybeEnableUploadButton();
    });

    $("#voteUploadButton").click(function() {
        var comment = $("#comment").val();
        $.ajax({
            type: "POST",
            url: "/vote",
            data: JSON.stringify({
                txid: txid,
                comment: comment,
                upvote: upvote
            }),
            success: function(data){
                createQRCode(qrv, data.paymentAddress);
                $("#votePaymentAmount").text("Send " + data.amountToPay + " BCH to the following address:");
                $("#votePaymentAddress").text(data.paymentAddress);
                $("#voteForm").hide();
                $("#votePaymentForm").show();
                $("#voteUploadButton").hide();
                var url = 'ws://'+ hostname + ':' + port + '/ws';
                var socket = new WebSocket(url);
                socket.onopen = function(event) {
                    socket.send(data.paymentAddress);
                };
                socket.onmessage = function(event) {
                    $("#votePaymentForm").hide();
                    $("#voteForm").hide();
                    $("#votePaymentReceived").show();
                    var audio = new Audio('/static/audio/coin-sound.mp3');
                    audio.play();
                    socket.close();
                };
            },
            error: function(result) {
                if (result.status === 403){
                    alert("Wait for transaction to confirm before commenting");
                    return
                }
                alert("Oops we messed up. Try again later.");
            },
            dataType: "json"
        });
    });
});

function clearVoteModal() {
    $("#commentRemainingChars").text("177 characters remaining");
    $("#comment").val("");
    $("#voteForm").show();
    $("#votePaymentForm").hide();
    $("#voteUploadButton").show();
    $("#votePaymentReceived").hide();
    $("#voteUp").removeClass("upvote");
    $("#voteDown").removeClass("downvote");
    $("#voteUp").addClass("thumb");
    $("#voteDown").addClass("thumb");
    qrv.clear();
}

function maybeEnableUploadButton() {
    var txt = $("#commentRemainingChars").text();
    var n = txt.indexOf(" ");
    var current = txt.substr(0, n);
    var remaining = parseInt(current);
    if (remaining >= 0) {
        $('#voteUploadButton').prop('disabled', false);
    } else {
        $('#voteUploadButton').prop('disabled', true);
    }
}