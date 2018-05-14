var cidLength = 0;
var cidValid = false;
var qrc;
$(function(){
    qrc = new QRCode(document.getElementById("qrcode"), "");
    $("#upload").click(function( event ) {
        event.preventDefault();
        $('#uploadModal').modal();
    });

    $('.dropdown-toggle').dropdown();
    $('.dropdown-item').click(function(event){
        $('#dropdownMenuButton').html(event.target.name);
        updateRemaining();
    });

    $("#uploadButton").click(function() {
        var desc = $("#description").val();
        var selectedCategory = $('#dropdownMenuButton').html();
        if (!selectedCategory.includes("Category")) {
            desc += '<meta name="category" content="' + selectedCategory + '"/>';
        }
        $.ajax({
            type: "POST",
            url: "/addfile",
            data: JSON.stringify({
                cid: $("#cidInput").val(),
                description: desc
            }),
            success: function(data){
                createQRCode(data.paymentAddress);
                $("#paymentAmount").text("Send " + data.amountToPay + " BCH to the following address:");
                $("#paymentAddress").text(data.paymentAddress);
                $("#uploadForm").hide();
                $("#paymentForm").show();
                $("#uploadButton").hide();
                var url = 'ws://'+ hostname + ':' + port + '/ws';
                var socket = new WebSocket(url);
                socket.onopen = function(event) {
                    socket.send(data.paymentAddress);
                };
                socket.onmessage = function(event) {
                    $("#paymentForm").hide();
                    $("#uploadForm").hide();
                    $("#paymentReceived").show();
                    var audio = new Audio('/static/audio/coin-sound.mp3');
                    audio.play();
                    socket.close();
                };
            },
            error: function(result) {
                alert("Oops we messed up. Try again later.");
            },
            dataType: "json"
        });
    });

    $("#cidInput").on("change keyup paste", function() {
        $.ajax({
            type: "POST",
            url: "/validatecid",
            data: JSON.stringify({
                cid: $("#cidInput").val()
            }),
            success: function(data){
                cidValid = data.valid;
                if (data.valid) {
                    if (cidLength == 0) {
                        var txt = $("#remainingChars").text();
                        var n = txt.indexOf(" ");
                        var current = txt.substr(0, n);
                        var remaining = parseInt(current) - data.length;
                        $("#remainingChars").text(remaining + " characters remaining");
                    }
                    cidLength = data.length;
                    $("#cidInput").css("color", "#495057");
                } else {
                    $("#cidInput").css("color", "red");
                }
                maybeEnableUploadButton();
            },
            dataType: "json"
        });
    });

    $("#description").on('change keyup paste', function() {
        updateRemaining();
    });
});

function updateRemaining(){
    var desc = $("#description").val();
    var selectedCategory = $('#dropdownMenuButton').html();
    if (!selectedCategory.includes("Category")) {
        desc += '<meta name="category" content="' + selectedCategory + '"/>';
    }
    var currentLenth = lengthInUtf8Bytes(desc);
    var remaining = 214 - cidLength - currentLenth;
    $("#remainingChars").text(remaining + " characters remaining");
    maybeEnableUploadButton();
}

function lengthInUtf8Bytes(str) {
    var m = encodeURIComponent(str).match(/%[89ABab]/g);
    return str.length + (m ? m.length : 0);
}

function clearModal() {
    $("#remainingChars").text("214 characters remaining");
    $("#description").val("");
    $("#cidInput").val("");
    $("#uploadForm").show();
    $("#paymentForm").hide();
    $("#uploadButton").show();
    $("#paymentReceived").hide();
    $('#dropdownMenuButton').html("Category");
    qrc.clear();
    cidLength = 0;
    maybeEnableUploadButton();
}

function maybeEnableUploadButton() {
    if (cidValid && $('#description').val().length > 0) {
        $('#uploadButton').prop('disabled', false);
    } else {
        $('#uploadButton').prop('disabled', true);
    }
}