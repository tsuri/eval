load("evaluations/graphs/eval.star", "m")

# there should be a way to define it somewhere else
_components = []

def component(
        name,
        components=[],
        inputs=[],
        outputs=[],
        config=None
):
    _components.append({
        "name": name,
        "components": components
    })

def value(
        name,
        type
):
    return {
        "name": name,
        "type": type
    }

def list(type):
    return {
        "type": type
    }

component(
    name = "digest"
)

component(
    name = "comparison",
    components = [
        component(
            name = "eval",
            components = [
                component(
                    name = "generate",
                    inputs = [
                        value(
                            name = "labels",
                            type = list(type="label")
                        )
                    ]
                ),
                component(
                    name = "snippet"
                ),
                component(
                    name = "aggregate_tables"
                )
            ]
        ),
        ":digest",
    ]
)

c = m
# load ("nodes", "action")

# # component has
# #
# # name
# # inputs
# # outputs
# # config
# # executor (things like cores, memory, gpu, etc...)
# for
# component(
#     name = "eval",
#     components = [
#         ":generate",
#         [":snippet" for i in generate.out],
#         ":aggregate",
#     ],
# )

# component(
#     name = "digest",
# )

# map_reduce(
#     name = "comparison",
#     map = ":eval",
#     reduce = ":digest",
#     inputs = [
#         value(
#             name = "git_references",
#             type = list("committish")
#         ),
#     ],
#     outputs = [
#         value(
#             name = "result",
#             type = json("schema"),
#             reference = "reduce.result",
#         )
#     ]
# )

# group(
#     name = "comparison",
#     components = [
#         ":eval"
#     ],
# )
