"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
// @ts-ignore
const core = __importStar(require("@aws/codecatalyst-adk-core"));
// @ts-ignore
const project = __importStar(require("@aws/codecatalyst-project"));
// @ts-ignore
const runSummaries = __importStar(require("@aws/codecatalyst-run-summaries"));
// @ts-ignore
const space = __importStar(require("@aws/codecatalyst-space"));
try {
    // Get inputs from the action
    const input_WhoToGreet = core.getInput('who-to-greet'); // Who are we greeting here
    console.log(input_WhoToGreet);
    const input_HowToGreet = core.getInput('how-to-greet'); // How to greet the person
    console.log(input_HowToGreet);
    // Interact with CodeCatalyst entities
    console.log(`Current CodeCatalyst space ${space.getSpace().name}`);
    console.log(`Current CodeCatalyst project ${project.getProject().name}`);
    // Action Code start
    // Set outputs of the action
    core.setOutput('greeting', `${input_HowToGreet} ${input_WhoToGreet}`);
    if (input_HowToGreet == 'fail') {
        runSummaries.RunSummaries.addRunSummary('Failed due to greeting type', runSummaries.RunSummaryLevel.ERROR);
    }
}
catch (error) {
    core.setFailed(`Action Failed, reason: ${error}`);
}
