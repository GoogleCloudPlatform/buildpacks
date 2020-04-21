namespace fs_function

open Google.Cloud.Functions.Framework
open Microsoft.AspNetCore.Http

type Function() =
    interface IHttpFunction with
        /// <summary>
        /// Logic for your function goes here.
        /// </summary>
        /// <param name="context">The HTTP context, containing the request and the response.</param>
        /// <returns>A task representing the asynchronous operation.</returns>
        member this.HandleAsync context =
            async {
                context.Response.WriteAsync "PASS" |> Async.AwaitTask |> ignore
            } |> Async.StartAsTask :> _
