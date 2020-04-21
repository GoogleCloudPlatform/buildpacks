Imports Google.Cloud.Functions.Framework
Imports Microsoft.AspNetCore.Http

Public Class CloudFunction
    Implements IHttpFunction

    ''' <summary>
    ''' Logic for your function goes here.
    ''' </summary>
    ''' <param name="context">The HTTP context, containing the request and the response.</param>
    ''' <returns>A task representing the asynchronous operation.</returns>
    Public Async Function HandleAsync(context As HttpContext) As Task Implements IHttpFunction.HandleAsync
        Await context.Response.WriteAsync("PASS")
    End Function
End Class
