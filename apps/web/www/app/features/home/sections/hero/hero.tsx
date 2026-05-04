import React, { useRef, useEffect } from 'react';
import { motion, type Variants } from 'motion/react';
import { ChevronRight, Database, Zap, Bell, ShieldCheck } from 'lucide-react';

interface LightningProps {
    hue?: number;
    xOffset?: number;
    speed?: number;
    intensity?: number;
    size?: number;
}

const Lightning: React.FC<LightningProps> = ({
    hue = 230,
    xOffset = 0,
    speed = 1,
    intensity = 1,
    size = 1,
}) => {
    const canvasRef = useRef<HTMLCanvasElement>(null);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) return;

        const resizeCanvas = () => {
            canvas.width = canvas.clientWidth;
            canvas.height = canvas.clientHeight;
        };
        resizeCanvas();
        window.addEventListener("resize", resizeCanvas);

        const gl = canvas.getContext("webgl");
        if (!gl) {
            console.error("WebGL not supported");
            return;
        }

        const vertexShaderSource = `
      attribute vec2 aPosition;
      void main() {
        gl_Position = vec4(aPosition, 0.0, 1.0);
      }
    `;

        const fragmentShaderSource = `
      precision mediump float;
      uniform vec2 iResolution;
      uniform float iTime;
      uniform float uHue;
      uniform float uXOffset;
      uniform float uSpeed;
      uniform float uIntensity;
      uniform float uSize;
      
      #define OCTAVE_COUNT 10

      // Convert HSV to RGB.
      vec3 hsv2rgb(vec3 c) {
          vec3 rgb = clamp(abs(mod(c.x * 6.0 + vec3(0.0,4.0,2.0), 6.0) - 3.0) - 1.0, 0.0, 1.0);
          return c.z * mix(vec3(1.0), rgb, c.y);
      }

      float hash11(float p) {
          p = fract(p * .1031);
          p *= p + 33.33;
          p *= p + p;
          return fract(p);
      }

      float hash12(vec2 p) {
          vec3 p3 = fract(vec3(p.xyx) * .1031);
          p3 += dot(p3, p3.yzx + 33.33);
          return fract((p3.x + p3.y) * p3.z);
      }

      mat2 rotate2d(float theta) {
          float c = cos(theta);
          float s = sin(theta);
          return mat2(c, -s, s, c);
      }

      float noise(vec2 p) {
          vec2 ip = floor(p);
          vec2 fp = fract(p);
          float a = hash12(ip);
          float b = hash12(ip + vec2(1.0, 0.0));
          float c = hash12(ip + vec2(0.0, 1.0));
          float d = hash12(ip + vec2(1.0, 1.0));
          
          vec2 t = smoothstep(0.0, 1.0, fp);
          return mix(mix(a, b, t.x), mix(c, d, t.x), t.y);
      }

      float fbm(vec2 p) {
          float value = 0.0;
          float amplitude = 0.5;
          for (int i = 0; i < OCTAVE_COUNT; ++i) {
              value += amplitude * noise(p);
              p *= rotate2d(0.45);
              p *= 2.0;
              amplitude *= 0.5;
          }
          return value;
      }

      void mainImage( out vec4 fragColor, in vec2 fragCoord ) {
          // Normalized pixel coordinates.
          vec2 uv = fragCoord / iResolution.xy;
          uv = 2.0 * uv - 1.0;
          uv.x *= iResolution.x / iResolution.y;
          // Apply horizontal offset.
          uv.x += uXOffset;
          
          // Adjust uv based on size and animate with speed.
          uv += 2.0 * fbm(uv * uSize + 0.8 * iTime * uSpeed) - 1.0;
          
          float dist = abs(uv.x);
          // Compute base color using hue.
          vec3 baseColor = hsv2rgb(vec3(uHue / 360.0, 0.7, 0.8));
          // Compute color with intensity and speed affecting time.
          vec3 col = baseColor * pow(mix(0.0, 0.07, hash11(iTime * uSpeed)) / dist, 1.0) * uIntensity;
          col = pow(col, vec3(1.0));
          fragColor = vec4(col, 1.0);
      }

      void main() {
          mainImage(gl_FragColor, gl_FragCoord.xy);
      }
    `;

        const compileShader = (
            source: string,
            type: number
        ): WebGLShader | null => {
            const shader = gl.createShader(type);
            if (!shader) return null;
            gl.shaderSource(shader, source);
            gl.compileShader(shader);
            if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
                console.error("Shader compile error:", gl.getShaderInfoLog(shader));
                gl.deleteShader(shader);
                return null;
            }
            return shader;
        };

        const vertexShader = compileShader(vertexShaderSource, gl.VERTEX_SHADER);
        const fragmentShader = compileShader(
            fragmentShaderSource,
            gl.FRAGMENT_SHADER
        );
        if (!vertexShader || !fragmentShader) return;

        const program = gl.createProgram();
        if (!program) return;
        gl.attachShader(program, vertexShader);
        gl.attachShader(program, fragmentShader);
        gl.linkProgram(program);
        if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
            console.error("Program linking error:", gl.getProgramInfoLog(program));
            return;
        }
        gl.useProgram(program);

        const vertices = new Float32Array([
            -1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1,
        ]);
        const vertexBuffer = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, vertexBuffer);
        gl.bufferData(gl.ARRAY_BUFFER, vertices, gl.STATIC_DRAW);

        const aPosition = gl.getAttribLocation(program, "aPosition");
        gl.enableVertexAttribArray(aPosition);
        gl.vertexAttribPointer(aPosition, 2, gl.FLOAT, false, 0, 0);

        const iResolutionLocation = gl.getUniformLocation(program, "iResolution");
        const iTimeLocation = gl.getUniformLocation(program, "iTime");
        const uHueLocation = gl.getUniformLocation(program, "uHue");
        const uXOffsetLocation = gl.getUniformLocation(program, "uXOffset");
        const uSpeedLocation = gl.getUniformLocation(program, "uSpeed");
        const uIntensityLocation = gl.getUniformLocation(program, "uIntensity");
        const uSizeLocation = gl.getUniformLocation(program, "uSize");

        const startTime = performance.now();
        const render = () => {
            resizeCanvas();
            gl.viewport(0, 0, canvas.width, canvas.height);
            gl.uniform2f(iResolutionLocation, canvas.width, canvas.height);
            const currentTime = performance.now();
            gl.uniform1f(iTimeLocation, (currentTime - startTime) / 1000.0);
            gl.uniform1f(uHueLocation, hue);
            gl.uniform1f(uXOffsetLocation, xOffset);
            gl.uniform1f(uSpeedLocation, speed);
            gl.uniform1f(uIntensityLocation, intensity);
            gl.uniform1f(uSizeLocation, size);
            gl.drawArrays(gl.TRIANGLES, 0, 6);
            requestAnimationFrame(render);
        };
        requestAnimationFrame(render);

        return () => {
            window.removeEventListener("resize", resizeCanvas);
        };
    }, [hue, xOffset, speed, intensity, size]);

    return <canvas ref={canvasRef} className="w-full h-full relative" />;
};


interface FeatureItemProps {
    name: string;
    value: string;
    position: string;
    icon: React.ReactNode;
}

const FeatureItem: React.FC<FeatureItemProps> = ({ name, value, position, icon }) => {
    return (
        <div className={`relative ${position} z-10 group transition-all duration-500 w-full lg:w-60`}>
            <motion.div
                whileHover={{ scale: 1.02, y: -2 }}
                className="relative flex items-center gap-3 p-3 md:p-4 rounded-2xl bg-gradient-to-br from-purple-500/10 to-transparent backdrop-blur-md border border-white/10 shadow-2xl overflow-hidden"
            >
                {/* Animated Background Gradient on Hover */}
                <div className="absolute inset-0 bg-linear-to-br from-purple-500/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500" />

                {/* Icon Container */}
                <div className="relative flex items-center justify-center w-8 h-8 md:w-10 md:h-10 rounded-xl bg-white/10 group-hover:bg-purple-500/20 transition-colors duration-500">
                    <div className="text-white/80 group-hover:text-white transition-colors duration-500">
                        {icon}
                    </div>
                </div>

                {/* Text Content */}
                <div className="relative flex flex-col">
                    <span className="font-semibold text-sm md:text-base text-white/90 group-hover:text-white transition-colors duration-500 whitespace-nowrap">
                        {name}
                    </span>
                    {value && (
                        <span className="text-[10px] md:text-xs text-white/50 group-hover:text-white/70 transition-colors duration-500 whitespace-nowrap">
                            {value}
                        </span>
                    )}
                </div>

                {/* Glow Effect */}
                <div className="absolute -inset-2 bg-purple-500/5 blur-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 -z-10" />
            </motion.div>
        </div>
    );
};

export const HeroSection: React.FC = () => {
    const lightningHue = 245;

    const containerVariants: Variants = {
        hidden: { opacity: 0 },
        visible: {
            opacity: 1,
            transition: {
                staggerChildren: 0.3,
                delayChildren: 0.2
            }
        }
    };

    const itemVariants: Variants = {
        hidden: { y: 20, opacity: 0 },
        visible: {
            y: 0,
            opacity: 1,
            transition: {
                duration: 0.5,
                ease: "easeOut"
            }
        }
    };

    return (
        <div className="relative w-full bg-background text-foreground overflow-hidden">
            {/* Main container with space for content */}
            <div className="2xl:mt-14 relative z-20 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 h-screen">

                {/* Main hero content */}
                <motion.div
                    variants={containerVariants}
                    initial="hidden"
                    animate="visible"
                    className="mt-28 relative z-30 flex flex-col items-center text-center max-w-4xl mx-auto "
                >
                    <motion.a
                        href="/download"
                        variants={itemVariants}
                        whileHover={{ scale: 1.05 }}
                        whileTap={{ scale: 0.95 }}
                        className="flex items-center space-x-2 px-4 py-2 bg-primary/5 hover:bg-primary/15 backdrop-blur-sm rounded-full text-sm mb-6 transition-all duration-300 group" // Reduced mb slightly
                    >
                        <span>Download the Desktop App</span>
                        <ChevronRight className='size-4 transform group-hover:translate-x-1 transition-transform duration-300' />
                    </motion.a>

                    <motion.h1
                        variants={itemVariants}
                        className="text-5xl md:text-7xl font-light mb-2 tracking-tight"
                    >
                        The Free Drop-in Replacement for LocalStack.
                    </motion.h1>

                    <motion.h2
                        variants={itemVariants}
                        className="text-3xl md:text-4xl pb-3 font-light bg-gradient-to-r from-gray-100 via-gray-200 to-gray-300 bg-clip-text text-transparent"
                    >
                        Run AWS services locally for free. <br /> <span className='text-2xl md:text-3xl'>Lightweight. Fast. Open Source.</span>
                    </motion.h2>

                    {/* Description */}
                    <motion.p
                        variants={itemVariants}
                        className="text-center text-gray-400 mb-9 max-w-2xl leading-relaxed font-light"
                    >
                        High-performance and lightweight local AWS emulator for <span className="font-normal text-foreground">S3</span>, <span className="font-normal text-foreground">DynamoDB</span>, <span className="font-normal text-foreground">SQS</span>, <span className="font-normal text-foreground">SNS</span>, and more. Includes an intuitive <span className="font-normal text-foreground">Desktop App</span> to manage and browse your local cloud resources from a single place. No docker required!
                    </motion.p>
                </motion.div>

                {/* Floating Widgets / Features */}
                <motion.div
                    variants={containerVariants}
                    initial="hidden"
                    animate="visible"
                    className="flex flex-col lg:flex-row items-center justify-between gap-6 z-[200] relative lg:absolute lg:top-[60%] xl:top-[58%] 2xl:top-[55%] lg:inset-x-0 mt-14 lg:mt-0 px-4 lg:px-8"
                >
                    <motion.div variants={itemVariants} className="lg:translate-y-20 w-full lg:w-auto">
                        <FeatureItem icon={<Database className="size-5" />} name="High-speed Storage" value="S3 & DynamoDB" position="" />
                    </motion.div>
                    <motion.div variants={itemVariants} className="lg:translate-y-4 w-full lg:w-auto">
                        <FeatureItem icon={<Zap className="size-5" />} name="Serverless Workflows" value="Lambda & SQS" position="" />
                    </motion.div>
                    <motion.div variants={itemVariants} className="lg:translate-y-4 w-full lg:w-auto">
                        <FeatureItem icon={<Bell className="size-5" />} name="Event-Driven Logic" value="SNS & EventBridge" position="" />
                    </motion.div>
                    <motion.div variants={itemVariants} className="lg:translate-y-20 w-full lg:w-auto">
                        <FeatureItem icon={<ShieldCheck className="size-5" />} name="Offline & Cost-Free" value="100% Local" position="" />
                    </motion.div>
                </motion.div>
            </div>

            {/* Background elements */}
            <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 1 }}
                className="absolute inset-0 z-0"
            >
                {/* Dark overlay */}
                <div className="absolute inset-0 bg-black/80"></div>

                {/* Glowing circle */}
                <div className="absolute top-[55%] left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] rounded-full bg-gradient-to-b from-purple-500/20 to-pink-600/10 blur-3xl"></div>

                {/* Central light beam - now using the state variable for hue */}
                <div className="absolute top-0 w-[100%] left-1/2 transform -translate-x-1/2 h-full">
                    <Lightning
                        hue={lightningHue} // Use the state variable here
                        xOffset={0}
                        speed={1.6}
                        intensity={0.7}
                        size={2.2}
                    />
                </div>

                {/* Planet/sphere */}
                <div className="z-10 absolute top-[55%] left-1/2 transform -translate-x-1/2 w-[600px] h-[600px] backdrop-blur-3xl rounded-full bg-[radial-gradient(circle_at_25%_90%,rgba(127,0,255,0.6)_15%,rgba(0,0,0,0.7)_70%,rgba(0,0,0,0.8)_100%)]"></div>
            </motion.div>
        </div>
    );
};
