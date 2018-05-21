$(function(){
    if (!more) {
        $('#nextPage').addClass("disabled");
    }
    if (page > 1) {
        $('#prevPage').removeClass("disabled");
    }
    $('#nextPage').click(function(){
        if (more) {
            window.location = "/search?query=" + query + "&page=" + (page + 1);
        }
    });
    $('#prevPage').click(function(){
        if (page-1 > 0) {
            window.location = "/search?query=" + query + "&page=" + (page - 1);
        }
    });
});