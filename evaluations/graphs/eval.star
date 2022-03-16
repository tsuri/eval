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

component(
    name = "eval"
)

def foo():
    def bar(x):
        return x+x
    return bar

f = foo()
m = f("Hallo") #foo()("hello")
